package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

func getData(id int64) string {
	fmt.Println("query...")
	time.Sleep(time.Second * 4)
	return fmt.Sprint(id)
}

/* 包装后的singleflight */
func doGetData(g *singleflight.Group, id int64) (string, error) {
	v, err, _ := g.Do("getData", func() (interface{}, error) {
		ret := getData(id)
		return ret, nil
	})
	return v.(string), err
}

func doChanGetData(ctx context.Context, g *singleflight.Group, id int64) (string, error) {
	ch := g.DoChan("getData", func() (interface{}, error) {
		ret := getData(id)
		return ret, nil
	})

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case ret := <-ch:
		return ret.Val.(string), ret.Err
	}

}

func main() {
	// 使用Do方法
	g := new(singleflight.Group) //单程航班 组
	go func() {
		s, err := doGetData(g, 10)
		fmt.Println(s, err)
	}()
	time.Sleep(time.Second)
	s, err := doGetData(g, 10)
	fmt.Println(s, err)

	// 使用Dochan方法
	go func() {
		doChanGetData(context.Background(), g, 10)
	}()
	time.Sleep(time.Second * 2)
	doChanGetData(context.Background(), g, 10)
}
