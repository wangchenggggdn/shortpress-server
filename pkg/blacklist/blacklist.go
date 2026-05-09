package blacklist

import (
	"bufio"
	"context"
	"os"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Blacklist 黑名单检查器
type Blacklist struct {
	words    map[string]bool
	mu       sync.RWMutex
	enabled  bool
	filePath string
}

// Config 黑名单配置
type Config struct {
	Enabled  bool   `yaml:"enabled"`
	FilePath string `yaml:"file_path"`
}

// NewBlacklist 创建新的黑名单检查器
func NewBlacklist(conf *viper.Viper) (*Blacklist, error) {
	bl := &Blacklist{
		words:    make(map[string]bool),
		enabled:  conf.GetBool("data.blacklist.enabled"),
		filePath: conf.GetString("data.blacklist.file_path"),
	}

	// 启动时加载词典
	if bl.enabled {
		if err := bl.LoadWords(context.Background()); err != nil {
			return nil, err
		}
	}

	return bl, nil
}

// IsEnabled 返回是否启用黑名单检查
func (bl *Blacklist) IsEnabled() bool {
	return bl.enabled
}

// LoadWords 加载黑名单词汇
func (bl *Blacklist) LoadWords(ctx context.Context) error {
	if !bl.enabled {
		return nil
	}

	file, err := os.Open(bl.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	bl.mu.Lock()
	defer bl.mu.Unlock()

	// 清空现有词典
	bl.words = make(map[string]bool)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" && !strings.HasPrefix(word, "#") { // 支持注释
			bl.words[strings.ToLower(word)] = true
		}
	}

	return scanner.Err()
}

// ContainsSensitiveWord 检查文本是否包含敏感词
func (bl *Blacklist) ContainsSensitiveWord(text string) (bool, string) {
	if !bl.enabled {
		return false, ""
	}

	bl.mu.RLock()
	defer bl.mu.RUnlock()

	if len(bl.words) == 0 {
		return false, ""
	}

	lowerText := strings.ToLower(text)
	for word := range bl.words {
		if lowerText == word {
			return true, word
		}
	}

	return false, ""
}

// GetWordCount 获取当前加载的敏感词数量
func (bl *Blacklist) GetWordCount() int {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return len(bl.words)
}
