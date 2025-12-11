# 版本 / 依赖 / Owner
- 版本：gorm v1.22.4；Go 1.17+（建议 1.19+）。
- 依赖：`gorm.io/gorm`、`gorm.io/driver/mysql`；日志适配使用团队 `pkg/log` + gorm Logger。
- 封装路径：`mfycommon/gorm`（规范）、业务侧仓储层引用此规范与自定义包装。
- 外部服务：MySQL 5.7+/8.0+。
- 最后更新时间：2024-05-xx。
- Owner：数据库规范小组（@架构负责人），评审人：服务域负责人；变更需双人 Review。
