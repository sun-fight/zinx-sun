package global

// Package utils 提供zinx相关工具类函数
// 包括:
//		全局配置
//		配置文件加载
//
// 当前文件描述:
// @Title  globalobj.go
// @Description  相关配置文件定义及加载方式
// @Author  Aceld - Thu Mar 11 10:32:29 CST 2019

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"time"

	"github.com/sun-fight/zinx-websocket/ziface"
)

type MysqlConfig struct {
	Path         string //服务器地址:端口
	Config       string // 高级配置
	Dbname       string // 数据库名
	Username     string // 数据库用户名
	Password     string // 数据库密码
	MaxIdleConns int    // 空闲中的最大连接数
	MaxOpenConns int    // 打开到数据库的最大连接数
	LogMode      string // 日志模式
}

type RedisConfig struct {
	DB       int    // redis的哪个数据库
	Addr     string // 服务器地址:端口
	Password string // 密码
}
type ZapConfig struct {
	Level         string `mapstructure:"level" json:"level" yaml:"level"`                           // 级别
	Format        string `mapstructure:"format" json:"format" yaml:"format"`                        // 输出
	Prefix        string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`                        // 日志前缀
	Director      string `mapstructure:"director" json:"director"  yaml:"director"`                 // 日志文件夹
	LinkName      string `mapstructure:"link-name" json:"linkName" yaml:"link-name"`                // 软链接名称
	ShowLine      bool   `mapstructure:"show-line" json:"showLine" yaml:"showLine"`                 // 显示行
	EncodeLevel   string `mapstructure:"encode-level" json:"encodeLevel" yaml:"encode-level"`       // 编码级
	StacktraceKey string `mapstructure:"stacktrace-key" json:"stacktraceKey" yaml:"stacktrace-key"` // 栈名
	LogInConsole  bool   `mapstructure:"log-in-console" json:"logInConsole" yaml:"log-in-console"`  // 输出控制台
}

var Redis *redis.Client
var MysqlRead *gorm.DB
var MysqlWrite *gorm.DB

/*
	存储一切有关Zinx框架的全局参数，供其他模块使用
	一些参数也可以通过 用户根据 zinx.json来配置
*/
type Obj struct {
	/*
		Server
	*/
	TCPServer ziface.IServer //当前Zinx的全局Server对象
	Host      string         //当前服务器主机IP
	TCPPort   int            //当前服务器主机监听端口号
	Name      string         //当前服务器名称
	// 详见[doublemsgid](https://github.com/sun-fight/zinx-websocket/tree/master/examples/doublemsgid)案例
	DoubleMsgID uint16 //(主子)双命令号模式(默认1单命令号模式)
	Env         string // develop production

	/*
		Zinx
	*/
	Version          string        //当前Zinx版本号
	MaxPacketSize    uint16        //读取数据包的最大值
	MaxConn          int           //当前服务器主机允许的最大链接个数
	WorkerPoolSize   uint32        //业务工作Worker池的数量
	MaxWorkerTaskLen uint32        //业务工作Worker对应负责的任务队列最大任务存储数量
	MaxMsgChanLen    uint32        //SendBuffMsg发送消息的缓冲最大长度
	HeartbeatTime    time.Duration //心跳间隔默认60秒,0=永不超时
	ConnReadTimeout  time.Duration //连接读取超时时间，0=永不超时,websocket连接状态已损坏且以后的所有读取都将返回错误。
	ConnWriteTimeout time.Duration //连接读取超时时间，0=永不超时,websocket连接状态已损坏且以后的所有读取都将返回错误。

	/*
		config file path
	*/
	ConfFilePath string

	/*
		zap
	*/
	ZapConfig ZapConfig

	/*
		数据库
	*/
	MysqlReadConfig MysqlConfig
	//可写操作数据库连接
	MysqlWriteConfig MysqlConfig
	//redis
	RedisConfig RedisConfig
}

// Object 定义一个全局的对象
var Object *Obj

//PathExists 判断一个文件是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//Reload 读取用户的配置文件
func (g *Obj) Reload() {
	if confFileExists, _ := PathExists(g.ConfFilePath); confFileExists != true {
		if Glog != nil {
			Glog.Error("Config File " + g.ConfFilePath + " is not exist!!")
		}
		return
	}

	v := viper.New()
	v.SetConfigFile(g.ConfFilePath)
	v.SetConfigType("json")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	v.WatchConfig()

	v.OnConfigChange(func(e fsnotify.Event) {
		Glog.Info("config file changed:" + e.Name)
		if err := v.Unmarshal(&g); err != nil {
			Glog.Error("配置文件更新失败", zap.Error(err))
		}
	})
	if err := v.Unmarshal(&g); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

}

/*
	提供init方法，默认加载
*/
func init() {
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "."
	}
	//初始化Object变量，设置一些默认值
	Object = &Obj{
		Name:             "ZinxServerApp",
		Version:          "V0.11",
		TCPPort:          8999,
		Host:             "0.0.0.0",
		Env:              "production",
		DoubleMsgID:      1,
		MaxConn:          12000,
		MaxPacketSize:    4096,
		ConfFilePath:     pwd + "/conf/zinx.json",
		WorkerPoolSize:   10,
		MaxWorkerTaskLen: 1024,
		MaxMsgChanLen:    1024,
		HeartbeatTime:    60,
		ConnReadTimeout:  60,
		ConnWriteTimeout: 60,
		ZapConfig: ZapConfig{
			Level:         "info",
			Format:        "console",
			Prefix:        "[zinx-websocket]",
			Director:      "log",
			LinkName:      "latest_log",
			ShowLine:      true,
			EncodeLevel:   "LowercaseColorLevelEncoder",
			StacktraceKey: "stacktrace",
			LogInConsole:  true,
		},
	}

	//NOTE: 从配置文件中加载一些用户配置的参数
	Object.Reload()
}
