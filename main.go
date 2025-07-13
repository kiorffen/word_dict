package main

import (
    "encoding/json"
    "fmt"
    "github.com/gin-gonic/gin"
    "github.com/jinzhu/gorm"
    _ "github.com/go-sql-driver/mysql"
    "golang.org/x/crypto/bcrypt"
    "net/http"
    "net/url"
    "strings"
)

var db *gorm.DB

type User struct {
    gorm.Model
    Username string `gorm:"unique"`
    Password string
}

type Word struct {
    gorm.Model
    UserID      uint    `json:"user_id"`
    Word        string  `json:"word"`
    Phonetic    string  `json:"phonetic"`
    Definition  string  `json:"definition"`
    AudioURL    string  `json:"audioURL"`
}

func initDB() {
    var err error
    db, err = gorm.Open("mysql", "root:haiyu198977@tcp(nps.tanghaiyu.com:8037)/word_dict?charset=utf8&parseTime=True&loc=Local")
    if err != nil {
        panic("Failed to connect to database")
    }
    
    // Auto migrate the schema
    db.AutoMigrate(&User{}, &Word{})

    // Create default user if not exists
    var user User
    if db.Where("username = ?", "haiyu").First(&user).RecordNotFound() {
        hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Tang777"), bcrypt.DefaultCost)
        db.Create(&User{
            Username: "haiyu",
            Password: string(hashedPassword),
        })
    }
}

func main() {
    initDB()
    defer db.Close()

    r := gin.Default()

    // Serve static files
    r.Static("/static", "./static")
    
    // Set custom delimiters for GO templates to avoid conflict with Vue.js
    r.Delims("[[", "]]")
    r.LoadHTMLGlob("templates/*")

    // Authentication middleware
    auth := func(c *gin.Context) {
        session := c.GetHeader("Authorization")
        if session == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }

    // Routes
    r.POST("/login", handleLogin)
    r.POST("/change-password", auth, handleChangePassword)
    
    // Word management routes
    r.GET("/words", auth, getWords)
    r.POST("/words", auth, addWord)
    r.PUT("/words/:id", auth, updateWord)
    r.DELETE("/words/:id", auth, deleteWord)

    // Serve the main page
    r.GET("/", func(c *gin.Context) {
        c.HTML(http.StatusOK, "index.html", nil)
    })

    r.Run(":8089")
}

func handleLogin(c *gin.Context) {
    var user User
    var input struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }

    if err := c.BindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"token": "session_" + user.Username})
}

func handleChangePassword(c *gin.Context) {
    var input struct {
        OldPassword string `json:"oldPassword"`
        NewPassword string `json:"newPassword"`
    }

    if err := c.BindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    username := c.GetHeader("Authorization")[8:] // Remove "session_" prefix
    var user User
    if err := db.Where("username = ?", username).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid old password"})
        return
    }

    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
    user.Password = string(hashedPassword)
    db.Save(&user)

    c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

func getWords(c *gin.Context) {
    username := c.GetHeader("Authorization")[8:]
    var user User
    db.Where("username = ?", username).First(&user)

    var words []Word
    db.Where("user_id = ?", user.ID).Find(&words)
    c.JSON(http.StatusOK, words)
}

func fetchWordInfo(word string) (phonetic string, audioURL string, err error) {
    // 首先尝试从 DictionaryAPI 获取数据
    resp, err := http.Get("https://api.dictionaryapi.dev/api/v2/entries/en/" + url.QueryEscape(word))
    if err != nil {
        return "", "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", "", fmt.Errorf("API error: %d", resp.StatusCode)
    }

    var result []struct {
        Phonetics []struct {
            Text  string `json:"text"`
            Audio string `json:"audio"`
        } `json:"phonetics"`
        Phonetic string `json:"phonetic"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", "", err
    }

    // 获取音标和音频URL
    if len(result) > 0 {
        // 优先使用 Phonetics 数组中的数据
        for _, p := range result[0].Phonetics {
            if p.Audio != "" && audioURL == "" {
                audioURL = p.Audio
            }
            if p.Text != "" && phonetic == "" {
                phonetic = p.Text
            }
        }
        
        // 如果 Phonetics 中没有音标，使用顶层的 phonetic 字段
        if phonetic == "" && result[0].Phonetic != "" {
            phonetic = result[0].Phonetic
        }
    }

    // 如果没有获取到音频URL，使用有道翻译的 TTS 服务作为备用
    if audioURL == "" {
        // 使用有道翻译的 TTS 服务，提供 mp3 格式音频
        audioURL = fmt.Sprintf("https://dict.youdao.com/dictvoice?audio=%s&type=2", url.QueryEscape(word))
    }

    // 确保音频URL是HTTPS的
    if audioURL != "" && !strings.HasPrefix(audioURL, "https://") {
        audioURL = "https://" + strings.TrimPrefix(audioURL, "http://")
    }

    return phonetic, audioURL, nil
}

func addWord(c *gin.Context) {
    username := c.GetHeader("Authorization")[8:]
    var user User
    db.Where("username = ?", username).First(&user)

    var word Word
    if err := c.BindJSON(&word); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    // 获取音标和发音URL
    phonetic, audioURL, err := fetchWordInfo(word.Word)
    if err != nil {
        // 如果获取失败，仍然继续保存单词，但不包含音标和发音
        fmt.Printf("Error fetching word info: %v\n", err)
    } else {
        word.Phonetic = phonetic
        word.AudioURL = audioURL
    }

    word.UserID = user.ID
    db.Create(&word)
    c.JSON(http.StatusOK, word)
}

func updateWord(c *gin.Context) {
    username := c.GetHeader("Authorization")[8:]
    var user User
    db.Where("username = ?", username).First(&user)

    var word Word
    if err := db.Where("id = ? AND user_id = ?", c.Param("id"), user.ID).First(&word).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Word not found"})
        return
    }

    var input Word
    if err := c.BindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    word.Word = input.Word
    word.Phonetic = input.Phonetic
    word.Definition = input.Definition
    word.AudioURL = input.AudioURL

    db.Save(&word)
    c.JSON(http.StatusOK, word)
}

func deleteWord(c *gin.Context) {
    username := c.GetHeader("Authorization")[8:]
    var user User
    db.Where("username = ?", username).First(&user)

    var word Word
    if err := db.Where("id = ? AND user_id = ?", c.Param("id"), user.ID).First(&word).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Word not found"})
        return
    }

    db.Delete(&word)
    c.JSON(http.StatusOK, gin.H{"message": "Word deleted successfully"})
}
