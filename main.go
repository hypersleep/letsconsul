package main

import "log"

func main() {
	log.Println("Loading configuration")

	app := &App{}

	err := app.config()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Registration in consul")

	err = app.register()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Loading application")

	err = app.start()
	if err != nil {
		log.Fatal(err)
	}
}