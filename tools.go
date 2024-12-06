package dba

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"github.com/google/uuid"
	"github.com/guregu/null/v5"
	"github.com/samber/lo"
	"golang.org/x/crypto/bcrypt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Time tools

func DiffDays(start, end time.Time) []time.Time {
	var dates []time.Time
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}
	return dates
}

func DiffDaysString(start, end time.Time) []string {
	return lo.Map[time.Time, string](DiffDays(start, end), func(item time.Time, index int) string {
		return item.Format(time.DateOnly)
	})
}

// Parse tools

func ParseNullFloat(s string) null.Float {
	if v, err := strconv.ParseFloat(s, 64); err != nil {
		return null.NewFloat(0, false)
	} else {
		return null.FloatFrom(v)
	}
}

func ParseNullInt(s string) null.Int {
	if v, err := strconv.Atoi(s); err != nil {
		return null.NewInt(0, false)
	} else {
		return null.IntFrom(int64(v))
	}
}

func ParseNullString(s, defaultValue string) null.String {
	s = strings.TrimSpace(s)
	if s == "" {
		return null.NewString(defaultValue, true)
	} else {
		return null.StringFrom(s)
	}
}

func ParseNullBool(s string) null.Bool {
	if v, err := strconv.ParseBool(s); err != nil {
		return null.NewBool(false, false)
	} else {
		return null.BoolFrom(v)
	}
}

// UUID tools

func NewUUID(upper bool, hyphen bool) string {
	s := uuid.NewString()
	if upper {
		s = strings.ToUpper(s)
	}
	if !hyphen {
		s = strings.ReplaceAll(s, "-", "")
	}
	return s
}

func NewUUIDToken() string {
	return NewUUID(true, false)
}

// JSON tools

func JSONStringify(v any, format ...bool) string {
	var (
		b   []byte
		err error
	)
	if len(format) > 0 && format[0] {
		b, err = json.MarshalIndent(v, "", "  ")
	} else {
		b, err = json.Marshal(v)
	}
	if err == nil {
		return string(b)
	}
	return ""
}

func JSONParse(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}

// Encode tools

func EncodeURIComponent(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	return r
}

func DecodeURIComponent(str string) string {
	if r, err := url.QueryUnescape(str); err == nil {
		return r
	}
	return str
}

// Random tools

func NewRandomNumber(length int) string {
	if length <= 0 {
		return ""
	}

	// 设置随机数生成器的种子
	rand.Seed(time.Now().UnixNano())

	// 第一位不能为0
	firstDigit := rand.Intn(9) + 1 // 1-9 之间的随机数

	// 生成剩余位数的随机数
	randomNumber := fmt.Sprintf("%d", firstDigit)
	for i := 1; i < length; i++ {
		randomNumber += fmt.Sprintf("%d", rand.Intn(10)) // 0-9 之间的随机数
	}

	return randomNumber
}

// Markdown tools

func MarkdownFindAndReplaceURLs(result string, keepText bool) (string, []string) {
	// 定义正则表达式模式，用于匹配标签内容
	regexPattern := `\[(.*?)\]\((.*?)\)`
	// 编译正则表达式
	re := regexp.MustCompile(regexPattern)
	// 查找所有匹配项
	matches := re.FindAllStringSubmatch(result, -1)
	var urls []string
	// 遍历所有匹配项
	for _, match := range matches {
		text := strings.TrimSpace(match[1])    // 获取匹配到的链接文本
		content := strings.TrimSpace(match[2]) // 获取匹配到的链接内容
		if content != "" {
			urls = append(urls, content)
		}
		// 进行文本替换
		if keepText && text != "" {
			result = strings.Replace(result, match[0], text, 1)
		} else {
			result = strings.Replace(result, match[0], "", 1)
		}
	}
	return result, urls
}

// File tools

type FileKind string

const (
	FileKindText  FileKind = "TEXT"
	FileKindImage FileKind = "IMAGE"
	FileKindVideo FileKind = "VIDEO"
	FileKindAudio FileKind = "AUDIO"
	FileKindFile  FileKind = "FILE"
)

type File struct {
	rawFilePath string
	ThumbID     null.String `mod:"缩略图ID"`
	ThumbPath   null.String `mod:"缩略图"`
	ThumbURL    null.String `mod:"缩略图"`
	Duration    null.Float  `mod:"音/视频时长（秒）"`

	UUID     string      `mod:"文件唯一ID"`
	FileKind FileKind    `mod:"文件类型"`
	MineType string      `mod:"文件Mine-Type"` // 如image/jpeg
	Name     string      `mod:"文件名称"`
	Path     string      `mod:"存储路径"`
	Ext      string      `mod:"文件扩展名"`
	URL      string      `mod:"下载路径"`
	Size     null.Int    `mod:"文件大小"`
	Sort     null.Int    `mod:"排序值"`
	MD5Sum   null.String `mod:"MD5校验码"`
	Width    null.Int    `mod:"宽度"`
	Height   null.Int    `mod:"高度"`

	Clazz     null.String `mod:"文件分类"`
	OwnerID   null.Int    `mod:"所属数据ID"`
	OwnerType null.String `mod:"所属数据类型"`
}

func (f *File) SetMD5Sum() *File {
	v, _ := GetFileMD5Sum(f.rawFilePath)
	f.MD5Sum = null.StringFrom(v)
	return f
}

type FilepathObject struct {
	Raw           string
	FileNameNoExt string
	FullFileName  string
	FileExt       string
	FileDir       string
}

func PathExists(p string) (bool, error) {
	_, err := os.Stat(p)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func EnsureDir(p string) error {
	exists, err := PathExists(p)
	if err != nil {
		return err
	}
	if !exists {
		return os.MkdirAll(p, os.ModePerm)
	}
	return nil
}

func ParseFilepath(fileURL string) (fo *FilepathObject) {
	fo = new(FilepathObject)
	fileURL = strings.TrimSpace(fileURL)
	if fileURL == "" {
		return
	}
	// 获取纯文件名（不含后缀）
	fileName := filepath.Base(fileURL)
	fileNameNoExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]

	// 获取完整文件名（含后缀）
	fullFileName := filepath.Base(fileURL)

	// 获取文件后缀
	fileExt := strings.ToLower(filepath.Ext(fileURL))

	// 获取文件路径
	fileDir := filepath.Dir(fileURL)
	return &FilepathObject{
		Raw:           fileURL,
		FileNameNoExt: fileNameNoExt,
		FullFileName:  fullFileName,
		FileExt:       fileExt,
		FileDir:       fileDir,
	}
}

func GetFileMD5Sum(filePath string) (string, error) {
	// 打开文件
	fd, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	// 使用 md5 包创建一个新的 MD5 散列
	hash := md5.New()

	// 将文件内容写入哈希
	if _, err := io.Copy(hash, fd); err != nil {
		return "", err
	}

	// 计算 MD5 校验和
	hashInBytes := hash.Sum(nil)

	// 将字节转换为十六进制字符串
	md5Checksum := hex.EncodeToString(hashInBytes)

	return md5Checksum, nil
}

func IsImage(filePath string) bool {
	ext := filepath.Ext(filePath)
	fileType := map[string]string{
		".jpg":  "image/jpeg",
		".jp2":  "image/jp2",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".cr2":  "image/x-canon-cr2",
		".tif":  "image/tiff",
		".bmp":  "image/bmp",
		".jxr":  "image/vnd.ms-photo",
		".psd":  "image/vnd.adobe.photoshop",
		".ico":  "image/vnd.microsoft.icon",
		".heif": "image/heif",
		".dwg":  "image/vnd.dwg",
		".exr":  "image/x-exr",
		".avif": "image/avif",
	}[ext]
	return fileType != ""
}

func IsVideo(filePath string) bool {
	ext := filepath.Ext(filePath)
	fileType := map[string]string{
		".mp4":  "video/mp4",
		".m4v":  "video/x-m4v",
		".mkv":  "video/x-matroska",
		".webm": "video/webm",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".wmv":  "video/x-ms-wmv",
		".mpg":  "video/mpeg",
		".flv":  "video/x-flv",
		".3gp":  "video/3gpp",
	}[ext]
	return fileType != ""
}

func IsAudio(filePath string) bool {
	ext := filepath.Ext(filePath)
	fileType := map[string]string{
		".mid":  "audio/midi",
		".mp3":  "audio/mpeg",
		".m4a":  "audio/mp4",
		".ogg":  "audio/ogg",
		".flac": "audio/x-flac",
		".wav":  "audio/x-wav",
		".amr":  "audio/amr",
		".aac":  "audio/aac",
		".aiff": "audio/x-aiff",
	}[ext]
	return fileType != ""
}

func buildNestedQuery(value any, prefix string) (string, error) {
	components := ""

	switch vv := value.(type) {
	case []any:
		for i, v := range vv {
			component, err := buildNestedQuery(v, prefix+"[]")

			if err != nil {
				return "", err
			}

			components += component

			if i < len(vv)-1 {
				components += "&"
			}
		}

	case map[string]any:
		length := len(vv)

		for k, v := range vv {
			childPrefix := ""

			if prefix != "" {
				childPrefix = prefix + "[" + url.QueryEscape(k) + "]"
			} else {
				childPrefix = url.QueryEscape(k)
			}

			component, err := buildNestedQuery(v, childPrefix)

			if err != nil {
				return "", err
			}

			components += component
			length -= 1

			if length > 0 {
				components += "&"
			}
		}

	case string:
		if prefix == "" {
			return "", fmt.Errorf("value must be a map[string]any")
		}

		components += prefix + "=" + url.QueryEscape(vv)

	default:
		components += prefix
	}

	return components, nil
}

// snowflake tools

var snowflakeNode *snowflake.Node

func init() {
	var err error
	snowflakeNode, err = snowflake.NewNode(rand.Int63n(1024))
	if err != nil {
		panic(fmt.Errorf("snowflake init failed: %w", err))
	}
}

func NextSnowflakeID() snowflake.ID {
	return snowflakeNode.Generate()
}

func NextSnowflakeIntID() int64 {
	return NextSnowflakeID().Int64()
}

func NextSnowflakeStringID() string {
	return NextSnowflakeID().String()
}

// bcrypt tools

func GenPassword(inputPassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(inputPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	outputPassword := string(hash)
	return outputPassword, err
}

func CheckPassword(inputPassword, targetPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(targetPassword), []byte(inputPassword))
	return err == nil
}

func MD5Str(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// ParseStruct recursively parses a struct or a pointer to a struct and returns a map[string]any.
func ParseStruct(data any) map[string]any {
	result := make(map[string]any)
	parseStruct(reflect.ValueOf(data), result)
	return result
}

// parseStruct is the recursive function that does the actual parsing.
func parseStruct(value reflect.Value, result map[string]any) {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return
	}

	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		fieldValue := value.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle anonymous fields (embedded structs)
		if field.Anonymous {
			parseStruct(fieldValue, result)
			continue
		}

		if fieldValue.IsZero() {
			continue
		}

		if fieldValue.Kind() == reflect.Struct {
			nestedResult := make(map[string]any)
			parseStruct(fieldValue, nestedResult)
			result[field.Name] = nestedResult
		} else {
			result[field.Name] = fieldValue.Interface()
		}
	}
}

func SplitAndTrimSpace(s, sep string, removeEmpty ...bool) []string {
	var re bool
	if len(removeEmpty) > 0 {
		re = removeEmpty[0]
	}

	var result []string
	for _, item := range strings.Split(s, sep) {
		item = strings.TrimSpace(item)
		if (re && item != "") || !re {
			result = append(result, item)
		}
	}
	return result
}

func TrimSpaceSlice(s []string) {
	for i, item := range s {
		s[i] = strings.TrimSpace(item)
	}
}

func Item2List(dst any) any {
	v := reflect.Indirect(reflect.ValueOf(dst))
	if k := v.Kind(); k != reflect.Array && k != reflect.Slice {
		s := reflect.MakeSlice(reflect.SliceOf(v.Addr().Type()), 0, 0)
		s = reflect.Append(s, v.Addr())
		if s.CanAddr() {
			return s.Addr().Interface()
		}
		return s.Interface()
	}
	return dst
}

func IsNilOrZero(a any) bool {
	if a == nil {
		return true
	}

	v := reflect.ValueOf(a)
	if !v.IsValid() {
		return true
	}

	// Handle pointers and interfaces
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
		if !v.IsValid() {
			return true
		}
	}

	// Use IsZero for Go versions >= 1.13
	if v.IsZero() {
		return true
	}

	return false
}
