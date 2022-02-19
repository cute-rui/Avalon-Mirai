package main

import (
    "github.com/fsnotify/fsnotify"
    "github.com/spf13/viper"
    "log"
    "os"
    "strings"
)

var Conf = viper.New()

func confInit() {
    Conf.SetConfigType("toml")
    Conf.SetConfigName("avalon-mirai")
    Conf.AddConfigPath(`./soft/avalon/config/`)
    Conf.SetDefault("service.addr", ":6212")
    Conf.SetDefault("mirai.server.addr", "192.168.31.176:8080")
    Conf.SetDefault("mirai.server.websocketServerSideSyncId", "-1")
    replacer := strings.NewReplacer(".", "_")
    Conf.SetEnvKeyReplacer(replacer)
    err := Conf.ReadInConfig()
    if _, ok := err.(viper.ConfigFileNotFoundError); ok {
        _, err := os.Create("./avalon-mirai.toml")
        if err != nil {
            log.Println(err)
            return
        }
    }
    
    err = Conf.WriteConfig()
    if err != nil {
        log.Println(err)
        return
    }
    
    Conf.WatchConfig()
    Conf.OnConfigChange(func(in fsnotify.Event) {
        err := Conf.ReadInConfig()
        if err != nil {
            log.Println(err)
        }
    })
}

func init() {
    confInit()
}
