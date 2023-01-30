package config

import "os"

var HttpPort = os.Getenv("HTTP_PORT")
var CoreAPIURL = os.Getenv("CORE_SERVICE_URL")
var QueryAPIURL = os.Getenv("QUERY_SERVICE_URL")
var UploadAPIURL = os.Getenv("UPLOAD_SERVICE_URL")
var BridgeAPIURL = os.Getenv("BRIDGE_SERVICE_URL")
var ClusterAPIURL = os.Getenv("CLUSTER_SERVICE_URL")
var UserAPIURL = os.Getenv("USER_SERVICE_URL")
var DatabaseAPIURL = os.Getenv("DB_SERVICE_URL")
var AutomationAPIURL = os.Getenv("AUTOMATION_SERVICE_URL")

// NOTE: currently unused
var MongoURL = os.Getenv("MONGO_URL")
var KafkaURL = os.Getenv("KAFKA_URL1")
