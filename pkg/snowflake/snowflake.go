package snowflake

import (
	"errors"
	"time"

	sf "github.com/bwmarrin/snowflake"
)

// 雪花算法生成id
// 本质上是为了获取一个全局唯一的ID
var (
	InvalidInitParamErr  = errors.New("snowflake初始化失败,无效的startTime或machineID")
	InvalidTimeFormatErr = errors.New("snowflake初始化失败,无效的startTime格式")
)

var node *sf.Node

// Init 雪花算法初始化配置
func Init(startTime string, machineID int64) (err error) {
	if len(startTime) == 0 || machineID <= 0 {
		return InvalidInitParamErr
	}
	var st time.Time
	st, err = time.Parse("2006-01-02", startTime)
	if err != nil {
		return InvalidTimeFormatErr
	}
	sf.Epoch = st.UnixNano() / 1000000
	node, err = sf.NewNode(machineID)
	return
}

// GenID 生成一个ID
func GenID() int64 {
	return node.Generate().Int64()
}
