package procfs

import (
	"reflect"
	"testing"
)

func TestParsePIDList(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []int
	}{
		{name: "多行 pid", in: "123\n456\n789\n", want: []int{123, 456, 789}},
		{name: "带空白", in: "  123 \n\t456\n", want: []int{123, 456}},
		{name: "跳过非数字行", in: "123\nPID\n456\n", want: []int{123, 456}},
		{name: "跳过非正数", in: "0\n-5\n123\n", want: []int{123}},
		{name: "空输出", in: "", want: nil},
		{name: "仅表头", in: "PID\n", want: nil},
		{name: "无换行结尾", in: "123", want: []int{123}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParsePIDList(c.in); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("ParsePIDList(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestParseIntField(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
		ok   bool
	}{
		{name: "普通", in: "123\n", want: 123, ok: true},
		{name: "带空白", in: "   456   ", want: 456, ok: true},
		{name: "多字段取末尾", in: "  1  789", want: 789, ok: true},
		{name: "空", in: "", want: 0, ok: false},
		{name: "非数字", in: "PID", want: 0, ok: false},
		{name: "非正数", in: "-1", want: 0, ok: false},
		{name: "零", in: "0", want: 0, ok: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ParseIntField(c.in)
			if ok != c.ok || got != c.want {
				t.Fatalf("ParseIntField(%q) = (%d, %v), want (%d, %v)", c.in, got, ok, c.want, c.ok)
			}
		})
	}
}

func TestIsZombieState(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "僵尸", in: "Z", want: true},
		{name: "僵尸带后缀", in: "Z+", want: true},
		{name: "睡眠", in: "S", want: false},
		{name: "运行", in: "R+", want: false},
		{name: "空白包裹", in: "  Z  ", want: true},
		{name: "空", in: "", want: false},
		{name: "其他前缀", in: "SZ", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsZombieState(c.in); got != c.want {
				t.Fatalf("IsZombieState(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestParseLSOFPath(t *testing.T) {
	const txtOut = `COMMAND   PID  USER   FD     TYPE  DEVICE  SIZE/OFF      NODE NAME
rick    12345  u1   txt     REG    1,4  17203618  1234567 /opt/rick/bin/rick
rick    12345  u1   cwd     DIR    1,4       1024        2 /workdir/rick
`
	cases := []struct {
		name string
		out  string
		fd   string
		want string
	}{
		{name: "取 txt 行", out: txtOut, fd: "txt", want: "/opt/rick/bin/rick"},
		{name: "取 cwd 行", out: txtOut, fd: "cwd", want: "/workdir/rick"},
		{
			name: "COMMAND 含空格且 NAME 含空格",
			out: "COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"my app   99 u  txt  REG  1,4  100  1 /opt/my app dir/bin",
			fd:   "txt",
			want: "/opt/my app dir/bin",
		},
		{
			// fd 值出现在 NAME 中且后接 REG，但后方容不下 TYPE/DEVICE/SIZE/NODE
			// + NAME → 结构约束拒绝，不得误取
			name: "fd 值出现在 NAME 尾部但不满足列约束",
			out: "COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"rick  1 u cwd DIR  1,4 100 1 /workdir txt REG",
			fd:   "txt",
			want: "",
		},
		{
			name: "列数不足（非 lsof 输出）",
			out:  "some random text txt REG",
			fd:   "txt",
			want: "",
		},
		{name: "无匹配行", out: txtOut, fd: "1u", want: ""},
		{name: "空输出", out: "", fd: "txt", want: ""},
		{name: "仅表头", out: "COMMAND  PID USER FD TYPE DEVICE SIZE/OFF NODE NAME", fd: "txt", want: ""},
		{
			name: "多行取首个匹配",
			out: "COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"a  1 u txt REG 1,4 1 1 /first/x\n" +
				"b  2 u txt REG 1,4 1 2 /second/y",
			fd:   "txt",
			want: "/first/x",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParseLSOFPath(c.out, c.fd); got != c.want {
				t.Fatalf("ParseLSOFPath(fd=%q) = %q, want %q", c.fd, got, c.want)
			}
		})
	}
}
