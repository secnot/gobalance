package config

import (
	"github.com/spf13/viper"
)

type CastFunc func(v *viper.Viper, key string) interface{}


func CastBool(v *viper.Viper, key string) interface{} {
	return v.GetBool(key)
}

func CastDuration(v *viper.Viper, key string) interface{} {
	return v.GetDuration(key)
}

func CastFloat64(v *viper.Viper, key string) interface{} {
	return v.GetFloat64(key)
}

func CastInt(v *viper.Viper, key string) interface{} {
	return v.GetInt(key)
}

func CastInt64(v *viper.Viper, key string) interface{} {
	return v.GetInt64(key)
}

func CastSizeInBytes(v *viper.Viper, key string) interface{} {
	return v.GetSizeInBytes(key)
}

func CastString(v *viper.Viper, key string) interface{} {
	return v.GetString(key)
}

func CastStringMap(v *viper.Viper, key string) interface{} {
	return v.GetStringMap(key)
}

func CastStringMapString(v *viper.Viper, key string) interface{} {
	return v.GetStringMapString(key)
}

func CastStringMapStringSlice(v *viper.Viper, key string) interface{} {
	return v.GetStringMapStringSlice(key)
}

func CastStringSlice(v *viper.Viper, key string) interface{} {
	return v.GetStringSlice(key)
}

func CastTime(v *viper.Viper, key string) interface {} {
	return v.GetTime(key)
}

