# drone-vault

drone vault plugin

## 配置参数
与 [drone-vault](https://github.com/drone/drone-vault) 配置参数一致，唯一不同是 `drone-vault` 的 `DRONE_DEBUG` 参数换成 `DEBUG_LEVEL`，`DEBUG_LEVEL`默认值为`info`，支持`debug`、`info`、`warning`、`error`、`fatal`五个参数。

## 使用区别
原有 `.drone.yml` 中 `vault` 保管 `secret` 获取方式
```ymal
---
kind: secret
name: username
get:
  path: drone/data/test
  name: username
```
现在多了 `vault` 版本号的支持，
现在 `.drone.yml` 中 `vault` 保管 `secret` 获取方式
```ymal
---
kind: secret
name: username
get:
  path: v2:drone/data/test
  name: username
```
可以在`path`前面添加 `v` 开头，跟上版本号，再以 `:`结尾加上 `path` 获取指定版本号的 `secret`
