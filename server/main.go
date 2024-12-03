package main

import (
	"fmt" //for formatted
	"log" //logging error messages
	"math" //mathematical functions
	"strconv"  // convert between strings and other types

	"github.com/gofiber/fiber/v2" //for routing
	"github.com/gofiber/fiber/v2/middleware/cors" //front-end applications hosted on different domains
	"github.com/joho/godotenv" //load environment variables
	"github.com/ramya/database-server/prisma/db" // database operations
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}
	// Database Connection
	client := db.NewClient()
	if err := client.Prisma.Connect(); err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}
	defer client.Prisma.Disconnect()

	app := fiber.New() // routing and HTTP requests
	app.Use(cors.New()) //avoids CORS issues

	// Populate products
	app.Post("/api/products/populate", func(c *fiber.Ctx) error {
		for i := 0; i < 50; i++ {
			_, err := client.Product.CreateOne(
				db.Product.Title.Set(fmt.Sprintf("Product %d", i)),
				db.Product.Description.Set("This is a dummy product description."),
				db.Product.Image.Set(fmt.Sprintf("http://lorempixel.com/200/200?%d", i)),
				db.Product.Price.Set(10+i),
			).Exec(c.Context())
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Failed to populate database"})
			}
		}
		return c.JSON(fiber.Map{"message": "Products populated successfully"})
	})

	// Fetch all products (Frontend)
	app.Get("/api/products/frontend", func(c *fiber.Ctx) error {
		products, err := client.Product.FindMany().Exec(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch products"})
		}
		return c.JSON(products)
	})

	// Fetch products with pagination and search (Backend)
	app.Get("/api/products/backend", func(c *fiber.Ctx) error {
		search := c.Query("s", "")
		sort := c.Query("sort", "asc")
		page, _ := strconv.Atoi(c.Query("page", "1"))
		perPage := 9
		offset := (page - 1) * perPage

		// Fetch products with pagination
		products, err := client.Product.FindMany(
			db.Product.Or(
				db.Product.Title.Contains(search),
				db.Product.Description.Contains(search),
			),
		).OrderBy(
			db.Product.Price.Order(db.SortOrder(sort)),
		).Take(perPage).Skip(offset).Exec(c.Context())

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch products"})
		}

		// Count the total number of matching products by using FindMany and checking the length
		totalProducts, err := client.Product.FindMany(
			db.Product.Or(
				db.Product.Title.Contains(search),
				db.Product.Description.Contains(search),
			),
		).Exec(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch products for count"})
		}

		total := len(totalProducts)
		return c.JSON(fiber.Map{
			"data":      products,
			"total":     total,
			"page":      page,
			"last_page": math.Ceil(float64(total) / float64(perPage)),
		})
	})

	log.Fatal(app.Listen(":8081"))
}
