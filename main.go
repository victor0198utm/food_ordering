package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/victor0198utm/restaurant_hall/models"
)

var w sync.WaitGroup

var m sync.Mutex

// Global vars

var restaurants []models.RestaurantDescription
var complex_order_id = 1
var ratings = []models.Ratings{}

// Food ordering endpoint: "/"
func getMenu(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")

	menus := models.Menus{}
	menus.Restaurants = len(restaurants)

	for _, rCopy := range restaurants {
		menu := models.Menu{rCopy.Name, rCopy.Menu_items, rCopy.Menu, rCopy.Rating}
		menus.Restaurants_data = append(menus.Restaurants_data, menu)
	}

	jsonResp, err := json.Marshal(menus)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}

	fmt.Println("Menus:", menus)

	w.Write(jsonResp)
	return
}

func registerRestaurant(w http.ResponseWriter, r *http.Request) {
	var new_restaurant models.RestaurantDescription
	err := json.NewDecoder(r.Body).Decode(&new_restaurant)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	restaurants = append(restaurants, new_restaurant)
	ratings = append(ratings, models.Ratings{})
}

func getRestaurantAddress(id int) string {
	for _, restaurant := range restaurants {
		if restaurant.Restaurant_id == id {
			return restaurant.Address
		}
	}

	return ""
}

func clientOrder(w http.ResponseWriter, r *http.Request) {
	clientOrderRequest := models.ClientOrderReq{}
	err := json.NewDecoder(r.Body).Decode(&clientOrderRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(clientOrderRequest)

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")

	// hall responses
	responsesToClient := []models.OrderV2Resp{}

	for _, orderReqCopy := range clientOrderRequest.Orders {
		restaurantAddress := getRestaurantAddress(orderReqCopy.Restaurant_id)
		fmt.Println("Order to restaurant: ", restaurantAddress)
		ov2r := models.OrderV2Req{
			orderReqCopy.Items,
			orderReqCopy.Priority,
			orderReqCopy.Max_wait,
			orderReqCopy.Created_time,
		}

		json_data, err_marshall := json.Marshal(ov2r)
		if err_marshall != nil {
			log.Fatal(err_marshall)
		}

		resp, err := http.Post("http://"+restaurants[orderReqCopy.Restaurant_id-1].Address+"/v2/order", "application/json",
			bytes.NewBuffer(json_data))
		if err != nil {
			log.Fatal(err)
		}

		orderV2Response := models.OrderV2Resp{}
		err = json.NewDecoder(resp.Body).Decode(&orderV2Response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Println(orderV2Response)

		responsesToClient = append(responsesToClient, orderV2Response)
	}

	// make client response payload
	clientOrderResponse := models.ClientOrderResp{}
	m.Lock()
	clientOrderResponse.Order_id = complex_order_id
	complex_order_id = complex_order_id + 1
	m.Unlock()
	orders := []models.OrderResp{}
	for _, orderRespCopy := range responsesToClient {
		restaurantAddress := getRestaurantAddress(orderRespCopy.Restaurant_id)

		orders = append(orders, models.OrderResp{
			orderRespCopy.Restaurant_id,
			restaurantAddress,
			orderRespCopy.Order_id,
			orderRespCopy.Estimated_waiting_time,
			orderRespCopy.Created_time,
			orderRespCopy.Registered_time,
		})
	}
	clientOrderResponse.Orders = orders

	jsonResp, err := json.Marshal(clientOrderResponse)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}

	w.Write(jsonResp)
	return
}

func add_rating(w http.ResponseWriter, r *http.Request) {
	review := models.Review{}
	err := json.NewDecoder(r.Body).Decode(&review)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ratings[review.Restaurant_id-1].Reviews = append(ratings[review.Restaurant_id-1].Reviews, review.Stars)
	n := len(ratings[review.Restaurant_id-1].Reviews)
	sum := 0
	rating := float64(0)
	if n != 0 {
		for i := 0; i < n; i++ {
			sum += ratings[review.Restaurant_id-1].Reviews[i]
		}
		rating = float64(sum) / float64(n)
	}

	m.Lock()
	restaurants[review.Restaurant_id-1].Rating = rating
	m.Unlock()
}

// Requests hadler
func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/menu", getMenu).Methods("GET")
	myRouter.HandleFunc("/register", registerRestaurant).Methods("POST")
	myRouter.HandleFunc("/order", clientOrder).Methods("POST")
	myRouter.HandleFunc("/rating", add_rating).Methods("POST")
	log.Fatal(http.ListenAndServe(":8011", myRouter))
}

func display() {
	for {
		time.Sleep(5000 * time.Millisecond)
		fmt.Println(restaurants, "\n")
	}
	w.Done()
}

func main() {

	w.Add(1)
	go display()

	handleRequests()

	w.Wait()
}
