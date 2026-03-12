package consts

// 业务状态码
const (
	CodeSuccess       = 200
	CodeBadRequest    = 400
	CodeUnauthorized  = 401
	CodeNotFound      = 404
	CodeInternalError = 500
)

// GA 状态码
const (
	GACodeSuccess = 200000
)

// Redis 数据类型
const (
	TypeString      = "string"
	TypeHash        = "hash"
	TypeList        = "list"
	TypeSet         = "set"
	TypeZSet        = "zset"
	TypeStream      = "stream"
	TypeJSON        = "ReJSON-RL"
	TypeBitmap      = "bitmap"
	TypeBitfield    = "bitfield"
	TypeHyperLogLog = "hyperloglog"
	TypeGeospatial  = "geospatial"
)
