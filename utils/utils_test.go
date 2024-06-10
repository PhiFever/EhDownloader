package utils

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
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
			if got := ToSafeFilename(tt.in); got != tt.want {
				t.Errorf("ToSafeFilename() = %v, want %v", got, tt.want)
			}
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
		wantErr bool
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			func() {
				file, err := os.Create(tt.args.filePath)
				if err != nil {
					panic(err)
				}
				defer func(file *os.File) {
					err := file.Close()
					if err != nil {
						panic(err)
					}
				}(file)

				// 创建一个 bufio.Writer 来帮助按行写入数据
				writer := bufio.NewWriter(file)
				// 循环写入多行数据
				for _, line := range tt.want {
					_, err := fmt.Fprintln(writer, line)
					if err != nil {
						t.Errorf("ReadListFile().WriteList error = %v, wantErr %v", err, tt.wantErr)
					}
				}

				// 刷新缓冲区并检查错误
				if err := writer.Flush(); err != nil {
					panic(err)
				}
			}()
			got, err := ReadListFile(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadListFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadListFile() got = %v, want %v", got, tt.want)
			}
			//删除文件
			err = os.Remove(tt.args.filePath)
			if err != nil {
				t.Errorf("ReadListFile() remove file error = %v", err)
			}
		})
	}
}

func TestGetFileTotal(t *testing.T) {
	type args struct {
		dirPath    string
		fileSuffix []string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "../test",
			args: args{
				dirPath:    "../test",
				fileSuffix: []string{".jpg"},
			},
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if total := GetFileTotal(tt.args.dirPath, tt.args.fileSuffix); total != tt.want {
				t.Errorf("GetFileTotal() = %v, want %v", total, tt.want)
			}
		})
	}
}
