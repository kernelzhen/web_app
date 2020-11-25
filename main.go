package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"web_app/data/mysql"
	"web_app/data/redis"
	"web_app/logger"
	"web_app/routes"
	"web_app/settings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("need config file.eg: wap_app config.yaml")
		return
	}
	//1 加载配置
	if err := settings.Init(os.Args[1]); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}
	fmt.Println(settings.Conf)
	fmt.Println(settings.Conf.LogConfig == nil)

	// 初始化日志
	if err := logger.Init(settings.Conf.LogConfig); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()
	zap.L().Debug("logger init sucess...")
	//3初始化mysql连接
	if err := mysql.Init(settings.Conf.MySQLConfig); err != nil {
		fmt.Printf("init mysql failed, err:%v\n", err)
		return
	}
	defer mysql.Close()

	//4 redis
	if err := redis.Init(settings.Conf.RedisConfig); err != nil {
		fmt.Printf("init redis failed, err:%v\n", err)
		return
	}
	defer redis.Close()
	// 5 注册路由
	r := routes.Setup(settings.Conf.Mode)

	//6.启动服务
	fmt.Println(settings.Conf.Port)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", settings.Conf.Port),
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen:%s\n", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zap.L().Info("shutDown Server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("server shutdown", zap.Error(err))
	}
	zap.L().Info("server exiting")
}
