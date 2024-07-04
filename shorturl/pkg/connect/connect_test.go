package connect

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestGet(t *testing.T) {
	convey.Convey("基础用例", t, func() {
		url := "https://github.com/smartystreets/goconvey?tab=readme-ov-file"
		got := Get(url)
		// assert
		convey.So(got, convey.ShouldEqual, true)
		convey.ShouldBeTrue(got)
	})
	convey.Convey("url不通过的示例", t, func() {
		url := `posts/Go/unit-test`
		got := Get(url)
		// assert
		convey.ShouldBeFalse(got)
	})
}
