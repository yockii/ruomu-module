package manager

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/jwt/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"github.com/yockii/ruomu-core/cache"
	"github.com/yockii/ruomu-core/config"
	"github.com/yockii/ruomu-core/shared"

	"github.com/yockii/ruomu-module/model"
)

func (m *Manager) checkAuthorization(injectInfo *model.ModuleInjectInfo) fiber.Handler {
	authorizationCode := strings.ToLower(injectInfo.AuthorizationCode)
	if authorizationCode == "" || authorizationCode == "anno" {
		return func(ctx *fiber.Ctx) error {
			return ctx.Next()
		}
	}

	return jwtware.New(jwtware.Config{
		SigningKey:    []byte(shared.JwtSecret),
		ContextKey:    "jwt-subject",
		SigningMethod: "HS256",
		TokenLookup:   "header:Authorization,cookie:token",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if err.Error() == "Missing or malformed JWT" {
				return c.Status(fiber.StatusBadRequest).SendString("无效的token信息")
			} else {
				return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired Authorization Token")
			}
		},
		SuccessHandler: func(c *fiber.Ctx) error {
			jwtToken := c.Locals("jwt-subject").(*jwt.Token)
			claims := jwtToken.Claims.(jwt.MapClaims)
			uid := claims[shared.JwtClaimUserId].(string)
			sid := claims[shared.JwtClaimSessionId].(string)
			tenantId, hasTenantId := claims[shared.JwtClaimTenantId].(string)

			conn := cache.Get()
			defer conn.Close()
			cachedUid, err := redis.String(conn.Do("GET", shared.RedisSessionIdKey+sid))
			if err != nil {
				if err != redis.ErrNil {
					logrus.Errorln(err)
				}
				return c.Status(fiber.StatusUnauthorized).SendString("token信息已失效")
			}
			if cachedUid != uid {
				return c.Status(fiber.StatusUnauthorized).SendString("token信息不正确")
			}

			// 判断是否有权限 1、读取用户的权限信息 2、判断是否有权限
			// 获取用户角色
			roleIds, _ := redis.Strings(conn.Do("GET", shared.RedisKeyUserRoles+uid))
			if len(roleIds) == 0 {
				callReq := map[string]interface{}{
					"userId": uid,
				}
				reqBs, _ := json.Marshal(callReq)

				var roleIds []string
				// 调用注入点获取用户角色信息
				for moduleName, injectCodes := range m.moduleInjectCodes {
					for _, code := range injectCodes {
						if code == shared.InjectCodeAuthorizationInfoByRoleId {
							bs, err := m.moduleExecMap[moduleName].InjectCall(shared.InjectCodeAuthorizationInfoByUserId, nil, reqBs)
							if err != nil {
								logrus.Errorln(err)
								continue
							}
							ai := new(shared.AuthorizationInfo)
							_ = json.Unmarshal(bs, ai)
							roleIds = append(roleIds, ai.RoleIds...)
							break
						}
					}
				}
				// 得到所有角色id列表，放入缓存
				for _, id := range roleIds {
					roleId := id
					conn.Do("SADD", shared.RedisKeyUserRoles+uid, roleId)
				}
			}
			// 设定有效期
			conn.Do("EXPIRE", shared.RedisKeyUserRoles+uid, 3*24*60*60)

			passAuthorize := false
			if authorizationCode == "user" {
				passAuthorize = true
			} else {
				// 检查角色ID列表对应的权限信息中是否有对应的code
				for _, id := range roleIds {
					roleId := id
					codes, _ := redis.Strings(conn.Do("GET", shared.RedisKeyRoleResourceCode+roleId))
					if len(codes) == 0 {
						// 不存在，从注入点获取
						callReq := map[string]interface{}{
							"roleId": roleId,
						}
						reqBs, _ := json.Marshal(callReq)

						for moduleName, injectCodes := range m.moduleInjectCodes {
							for _, code := range injectCodes {
								if code == shared.InjectCodeAuthorizationInfoByRoleId {
									bs, err := m.moduleExecMap[moduleName].InjectCall(shared.InjectCodeAuthorizationInfoByRoleId, nil, reqBs)
									if err != nil {
										logrus.Errorln(err)
										continue
									}
									ai := new(shared.AuthorizationInfo)
									_ = json.Unmarshal(bs, ai)
									codes = append(codes, ai.ResourceCodes...)
									break
								}
							}
						}
						for _, c := range codes {
							code := c
							conn.Do("SADD", shared.RedisKeyRoleResourceCode+roleId, code)
						}
					}

					// 增加有效期
					conn.Do("EXPIRE", shared.RedisKeyRoleResourceCode+roleId, 3*24*60*60)

					for _, c := range codes {
						code := c
						if code == injectInfo.AuthorizationCode {
							passAuthorize = true
							break
						} else if strings.HasPrefix(injectInfo.AuthorizationCode, code+":") {
							passAuthorize = true
							break
						}
					}

					if passAuthorize {
						break
					}
				}
			}
			if !passAuthorize {
				// 未通过校验，无权限
				return c.Status(fiber.StatusForbidden).SendString("无访问权限")
			}

			c.Locals(shared.JwtClaimUserId, uid)
			if hasTenantId {
				c.Locals(shared.JwtClaimTenantId, tenantId)
			}

			// token续期
			_, _ = conn.Do("EXPIRE", shared.RedisSessionIdKey+sid, config.GetInt("userTokenExpire"))

			return c.Next()
		},
	})
}
