# Core 接口字段设计

## 响应字段

业务方法返回 SDK 强类型对象。需要额外字段时，通过 `JMService.WithResponse(target)` 取得调用副本，再调用同一个业务方法。`target` 是与响应 JSON 结构对应的非空指针，只需声明所需字段；嵌套字段使用嵌套结构体和 JSON 标签。

副本保留认证、请求头和连接配置，绑定不会修改原 service。副本的每次 JSON 响应都同时解析到强类型返回值和 `target`；复用副本会继续写入同一目标。分页方法的目标按每页响应依次解析，不自动汇总；并发调用使用各自的副本和目标。注册时需要额外响应字段，同样通过副本调用 `RegisterTerminalWithOptions`。

字段遵循 `encoding/json` 的规则：未声明字段忽略，未传递字段保持目标原值，新建对象中的值为零值或 `nil`。可选字段使用指针区分未传递与显式零值；`null` 将指针、切片、映射和接口置为 `nil`，对其他普通值类型不作修改。

两个解析目标都会尝试处理。类型不匹配时返回错误，已成功解析的字段保留；非法 JSON、无效目标和 HTTP 错误也会返回错误，调用方根据错误决定是否继续使用结果。

SDK 不在模型或客户端中保存原始正文，只保留声明的字段。调用方若选择完整映射或 `json.RawMessage` 作为目标，相应数据由调用方持有。底层 `httplib.Client` 提供相同的 `WithResponse(target)` 入口。

## Token 新增字段的调用方式

1. 下游定义 `extra` 结构体，用 JSON 标签声明需要读取的字段。结构从响应根对象开始对应；位于 `connect_options` 内的字段放在对应的嵌套结构体中。
2. 调用 `svc.WithResponse(&extra).GetConnectTokenInfo(tokenID, expireNow)`。需要公钥或其他取密选项时，在同一绑定方式后调用 `GetConnectTokenInfoWithPublicKey` 或 `GetConnectTokenInfoWithOptions`。
3. SDK 已有字段从返回的 `token` 读取，新增字段从 `extra` 读取。仅需已有字段时，直接调用业务方法。

SDK 使用同一份取密响应解析两个目标，不增加对 Core 的请求次数，也不把完整 JSON 保存在 Token 对象中。新增字段由下游声明并使用，无需为每个字段升级 SDK。

每次调用新建 `extra`，可选字段使用指针：字段缺失或为 `null` 时为 `nil`，业务可以跳过。调用返回错误时，应先判断错误，再决定是否使用已经解析成功的字段。

## 请求字段

`SuperConnectTokenReq`、`ConnectTokenSecretOptions`、`TerminalRegistrationOptions` 的 `ExtraFields` 由相应服务方法合并进 JSON 请求体。已序列化输出的强类型字段优先，未输出的可选字段可以由 `ExtraFields` 补充。`ExtraFields` 本身不参与结构体的 JSON 序列化，`Params` 只用于查询参数。

`SuperConnectTokenReq.ConnectOptions` 使用 `map[string]any` 传递连接选项；可选布尔参数使用指针，区分未传递与显式 `false`。个人凭据的所有者认证、版本和权限检查由 Core 执行。
