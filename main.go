package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	yaml "gopkg.in/yaml.v2"
)

type Reservation struct {
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	ID              string    `json:"reservation_id"`
	State           string    `json:"reservation_state"`
	VehicleDistance int       `json:"vehicle_distance"`
	StateUpdateTime time.Time `json:"state_upatde_time"`
}

type ChargerInfo struct {
	Location struct {
		Latitude   float32 `json:"lat" yaml:"latitude"`
		Longtitude float32 `json:"lng" yaml:"longtitude"`
	} `json:"location" yaml:"location"`
	UUID string `json:"uuid" yaml:"uuid"`
	Port int    `json:"port" yaml:"port"`
}

const serverUrl = "http://10.100.32.197:5012/api/chargers/post"
const config_path = "config.yml"

var reservations map[string]Reservation

var chargerInfo = ChargerInfo{}

func init() {
	reservations = make(map[string]Reservation)
}

func register() error {
	log.Printf("Registering")
	postJSON, err := json.Marshal(&chargerInfo)
	if err != nil {
		log.Printf("Error: %s", err)
	}
	log.Printf("Postring config: %s", postJSON)
	req, _ := http.NewRequest("POST", serverUrl, bytes.NewBuffer(postJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Println("response Status:", resp.Status)
	log.Println("response Headers:", resp.Header)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Println("response Body:", string(body))
	if resp.StatusCode != 200 {
		return errors.New("Status code is not 200")
	}
	log.Printf("Registration done")
	return nil
}

func registerLoop() {
	if err := register(); err != nil {
		log.Printf("Error registering: %#v", err.Error())
		time.Sleep(10 * time.Second)
		registerLoop()
	}
}

func serverReserveCreation(c *gin.Context) {
	var reservation Reservation
	if err := c.ShouldBindJSON(&reservation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reservations[reservation.ID] = reservation
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

func serverReserveDeletion(c *gin.Context) {
	var reservation Reservation
	if err := c.ShouldBindJSON(&reservation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, ok := reservations[reservation.ID]; ok {
		delete(reservations, reservation.ID)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"status": fmt.Sprintf("Reservation %s not found", reservation.ID)})
	}
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

func main() {
	content, err := ioutil.ReadFile(config_path)
	if err != nil {
		log.Fatalf("Problem reading configuration file: %v", err)
	}
	err = yaml.Unmarshal(content, &chargerInfo)
	if err != nil {
		log.Fatalf("Error parsing configuration file: %v", err)
	}
	router := gin.Default()
	router.GET("/config", func(c *gin.Context) {
		c.JSON(200, chargerInfo)
	})
	router.GET("/reservations", func(c *gin.Context) {
		c.JSON(200, reservations)
	})
	router.POST("/reserve", serverReserveCreation)
	router.DELETE("/reserve", serverReserveDeletion)

	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		panic(err)
	}
	log.Println("listening on", listener.Addr().String())
	_, portString, _ := net.SplitHostPort(listener.Addr().String())
	chargerInfo.Port, _ = strconv.Atoi(portString)
	go registerLoop()

	panic(http.Serve(listener, router))
}
