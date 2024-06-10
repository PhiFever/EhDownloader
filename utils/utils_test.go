package utils

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestToSafeFilename(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "case1",
			in:   `[sfs]\24r/f4?*<q>|:`,
			want: `[sfs]_24r_f4___q___`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ToSafeFilename(tt.in))
		})
	}
}

func TestReadListFile(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "case1",
			args: args{
				filePath: "list.txt",
			},
			want: []string{
				"https://e-hentai.org/g/1111111/1111111111/",
				"https://e-hentai.org/g/2222222/2222222222/",
				"https://e-hentai.org/g/3333333/3333333333/",
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			func() {
				file, _ := os.Create(tt.args.filePath)
				defer file.Close()

				// 创建一个 bufio.Writer 来帮助按行写入数据
				writer := bufio.NewWriter(file)
				// 循环写入多行数据
				for _, line := range tt.want {
					fmt.Fprintln(writer, line)
				}

				// 刷新缓冲区并检查错误
				writer.Flush()
			}()
			got, err := ReadListFile(tt.args.filePath)
			if tt.wantErr(t, err) {
				assert.Equal(t, tt.want, got)
			}
			//删除文件
			os.Remove(tt.args.filePath)
		})
	}
}
