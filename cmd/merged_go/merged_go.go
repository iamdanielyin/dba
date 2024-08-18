package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	// 设置目标目录和输出文件路径
	dir := "/Users/danielyin/Projects/github.com/iamdanielyin/dba"
	outputFile := "./output/merged_output.go"

	// 获取目录下所有的 Go 文件
	goFiles, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		fmt.Println("Error reading Go files:", err)
		return
	}

	// 排序文件，确保包含 package 声明的文件在前
	sort.SliceStable(goFiles, func(i, j int) bool {
		return goFiles[i] < goFiles[j]
	})

	// 存储包名和导入语句
	var packageName string
	importSet := map[string]bool{}
	var otherCodeLines []string

	// 读取所有文件内容
	for _, file := range goFiles {
		fmt.Println("Processing:", file)

		f, err := os.Open(file)
		if err != nil {
			fmt.Println("Error opening file:", err)
			continue
		}
		defer f.Close()

		reader := bufio.NewReader(f)
		insideImportBlock := false

		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				fmt.Println("Error reading file:", err)
				break
			}

			trimmedLine := strings.TrimSpace(line)

			if strings.HasPrefix(trimmedLine, "package ") && packageName == "" {
				packageName = trimmedLine
			} else if strings.HasPrefix(trimmedLine, "import (") {
				insideImportBlock = true
			} else if insideImportBlock {
				if trimmedLine == ")" {
					insideImportBlock = false
				} else if trimmedLine != "" {
					importSet[trimmedLine] = true
				}
			} else if strings.HasPrefix(trimmedLine, "import ") && !insideImportBlock {
				// 处理单行 import 并添加引号
				singleImport := strings.Trim(trimmedLine, `import "`)
				importPath := strings.Trim(singleImport, `"`)
				importSet[`"`+importPath+`"`] = true
			} else if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "package ") && !strings.HasPrefix(trimmedLine, "import ") {
				// 收集其他代码行
				otherCodeLines = append(otherCodeLines, line)
			}

			if err == io.EOF {
				break
			}
		}
	}

	// 创建输出文件
	if err := os.MkdirAll(filepath.Dir(outputFile), os.ModePerm); err != nil {
		fmt.Println("Error creating output directory:", err)
		return
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)

	// 写入 package 声明
	if packageName != "" {
		writer.WriteString(packageName + "\n\n")
	}

	// 写入 import 块
	if len(importSet) > 0 {
		writer.WriteString("import (\n")
		for imp := range importSet {
			writer.WriteString("\t" + imp + "\n")
		}
		writer.WriteString(")\n\n")
	}

	// 写入其他代码
	for _, line := range otherCodeLines {
		writer.WriteString(line)
	}

	writer.Flush()
	fmt.Println("All files have been merged into:", outputFile)
}
