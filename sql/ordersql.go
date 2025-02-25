package sql

import (
	"purchasing/config"
	"purchasing/handler"
	"purchasing/models"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func orderdeletesql(order int, user models.User) (message models.Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Deleting order ", order, "...")

	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr)
	}

	//Build the Query
	newquery := "DELETE FROM `orderskus` WHERE ordernum = ?"
	rows, err := config.DB.Query(newquery, order)
	rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}

	//Build the Query
	newquery = "DELETE FROM `orders` WHERE ordernum = ?"
	rows, err = config.DB.Query(newquery, order)
	rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}

	message.Success = true
	message.Title = "Success"
	message.Body = "Successfully deleted order " + strconv.Itoa(order)
	return message
}

func orderskuadd(order int, sku string, user models.User) (message models.Message) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Info("Inserting SKU/Order: ", sku, "/", order)

	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr)
	}
	//Build the Query
	newquery := "REPLACE INTO `orderskus`(`ordernum`, `sku_internal`) VALUES (?,?)"

	rows, err := config.DB.Query(newquery, order, sku)
	rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}

	message.Body = "Successfully inserted SKU " + sku
	message.Success = true
	return message
}

func orderlookup(ordernum int, user models.User) (message models.Message, orders []models.Order) {
	//Debug
	log.WithFields(log.Fields{"username": user.Username}).Debug("Getting Order: ", strconv.Itoa(ordernum))

	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr), orders
	}
	//Build the Query
	newquery := "SELECT ordernum,trackingnum,comments,manufacturer,status FROM `orders` WHERE ordernum = ?"

	orderrows, err := config.DB.Query(newquery, ordernum)
	if err != nil {
		return handler.Handleerror(pingErr), orders
	}
	defer orderrows.Close()
	log.WithFields(log.Fields{"username": user.Username}).Debug("Orderrows: ", orderrows)
	//Pull Data
	for orderrows.Next() {
		var r models.Order
		err := orderrows.Scan(&r.Ordernum, &r.Tracking, &r.Comments, &r.Manufacturer, &r.Status)
		if err != nil {
			return handler.Handleerror(pingErr), orders
		}
		//Build the Query for the skus in the order
		newquery := "SELECT a.sku_internal,`manufacturer_code`,`sku_manufacturer`,`product_option`,`processing_request`,`sorting_request`,`unit`,`unit_price`,`Currency`,`order_qty`,`modified`,`reorder`,`inventory_qty`,season FROM orderskus a left join skus b on a.sku_internal = b.sku_internal WHERE a.ordernum = ?"
		skurows, err := config.DB.Query(newquery, r.Ordernum)
		if err != nil {
			return handler.Handleerror(pingErr), orders
		}
		log.WithFields(log.Fields{"username": user.Username}).Debug("SKUrows: ", skurows)
		var skus []models.Product
		defer skurows.Close()
		for skurows.Next() {
			var r models.Product
			err := skurows.Scan(&r.SKU, &r.Manufacturer, &r.ManufacturerPart, &r.Description, &r.ProcessRequest, &r.SortingRequest, &r.Unit, &r.UnitPrice, &r.Currency, &r.OrderQty, &r.Modified, &r.Reorder, &r.InventoryQty, &r.Season)
			if err != nil {
				return handler.Handleerror(pingErr), orders
			}
			skus = append(skus, r)
		}
		r.Products = skus
		log.WithFields(log.Fields{"username": user.Username}).Debug("SKUS: ", skus)
		//Append to the orders
		orders = append(orders, r)
	}

	return message, orders
}

func orderupdatesql(order int, tracking string, comment string, status string, user models.User) (message models.Message) {
	//Test Connection
	pingErr := config.DB.Ping()
	if pingErr != nil {
		config.DB, message = config.Opendb()
		return handler.Handleerror(pingErr)
	}

	//Build the Query
	log.WithFields(log.Fields{"username": user.Username}).Debug("Building Query...")
	newquery := "UPDATE `orders` SET `trackingnum`=?,`comments`=?,`status`=? WHERE ordernum = ?"

	//Run Query
	rows, err := config.DB.Query(newquery, tracking, comment, status, order)
	defer rows.Close()
	if err != nil {
		return handler.Handleerror(err)
	}
	message.Body = "Successfully updated order " + strconv.Itoa(order)
	message.Success = true
	//Logging
	log.WithFields(log.Fields{"username": user.Username}).Info("Updated Order ", strconv.Itoa(order))
	return message
}
