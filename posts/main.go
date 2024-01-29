package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Post struct {
	Id           uint                `json:"id"`
	Title        string              `json:"title"`
	Description  string              `json:"description"`
	Comments     []map[string]string `json:"comments" gorm:"-" default:"[]"`
	CommentsJson string              `json:"-"`
}

type Comment struct {
	Text   string `json:"text"`
	PostID uint
}

func main() {
	db, err := gorm.Open(mysql.Open("root:@tcp(127.0.0.1:3306)/posts-microservices"), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	db.AutoMigrate(Post{})

	app := fiber.New()

	app.Use(cors.New())

	app.Get("/api/posts", func(c *fiber.Ctx) error {
		var posts []Post

		db.Find(&posts)

		for i, post := range posts {
			var comments []map[string]string

			json.Unmarshal([]byte(post.CommentsJson), &comments)

			posts[i].Comments = comments
		}

		return c.JSON(posts)
	})

	app.Post("/api/posts", func(c *fiber.Ctx) error {
		var post Post

		if err := c.BodyParser(&post); err != nil {
			return err
		}

		db.Create(&post)

		return c.JSON(post)
	})

	app.Post("/api/posts/:id/comments", func(c *fiber.Ctx) error {
		id, _ := strconv.Atoi(c.Params("id"))

		post := Post{
			Id: uint(id),
		}

		db.Model(Post{}).First(&post)

		var body map[string]string

		if err := c.BodyParser(&body); err != nil {
			return err
		}

		var comments []map[string]string

		json.Unmarshal([]byte(post.CommentsJson), &comments)

		comments = append(comments, map[string]string{
			"text": body["text"],
		})

		commentsJson, _ := json.Marshal(comments)

		db.Model(post).Where("id = ?", id).Update("comments_json", commentsJson)

		return c.JSON(post)
	})

	go Cron()

	app.Listen(":3000")
}

func Cron() {
	postDB, err := gorm.Open(mysql.Open("root:@tcp(127.0.0.1:3306)/posts-microservices"), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	commentDB, err := gorm.Open(mysql.Open("root:@tcp(127.0.0.1:3306)/comments-microservices"), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(time.Minute)

		var comments []Comment

		commentDB.Find(&comments)

		var posts []Post

		postDB.Find(&posts)

		for _, post := range posts {
			var postComments []map[string]string

			json.Unmarshal([]byte(post.CommentsJson), &postComments)

			var filteredcomments []map[string]string

			for _, comment := range comments {
				if comment.PostID == post.Id {
					filteredcomments = append(filteredcomments, map[string]string{
						"text": comment.Text,
					})
				}
			}

			if len(postComments) < len(filteredcomments) {
				commentsJson, _ := json.Marshal(filteredcomments)

				postDB.Model(post).Where("id = ?", post.Id).Update("comments_json", commentsJson)
			}
		}
	}
}
