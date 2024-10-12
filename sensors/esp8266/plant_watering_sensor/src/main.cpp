#include <Arduino.h>
#include <ESP8266WiFi.h>
#include <ESP8266WebServer.h>
#include <string.h>

/*Put your SSID & Password*/
const char* ssid = "HN-2-2,4G";
const char* password = "Szczecinnadmorzem1";

ESP8266WebServer server(80);

int sensorPin = A0;  // input pin for the potentiometer
int digitalValue = 0;// variable to store the value coming from the sensor
float analogOut = 0.00;

void handle_take_reading();
void handle_ledon();
void handle_ledon();
void handle_ledoff();
void handle_NotFound();

void setup() {
  Serial.begin(115200);
  delay(100);
  pinMode(LED_BUILTIN, OUTPUT);

  Serial.println("Connecting to ");
  Serial.println(ssid);

  //connect to your local wi-fi network
  WiFi.begin(ssid, password);

  //check wi-fi is connected to wi-fi network
  while (WiFi.status() != WL_CONNECTED) {
  delay(1000);
  Serial.print(".");
  }
  Serial.println("");
  Serial.println("WiFi connected..!");
  Serial.print("Got IP: ");  Serial.println(WiFi.localIP());

  server.on("/ledon", handle_ledon);
  server.on("/ledoff", handle_ledoff);
  server.on("/take_reading", handle_take_reading);
  server.onNotFound(handle_NotFound);

  server.begin();
  Serial.println("HTTP server started");
}

void loop() {
  server.handleClient();
}

void handle_take_reading() {
  digitalValue = analogRead(sensorPin);// read the value from the analog channel
  analogOut = (digitalValue * 5.00)/1023.00;
  server.send(200, "text/plain", String(analogOut));
}

void handle_ledon() {
  digitalWrite(LED_BUILTIN, LOW);
  server.send(200, "text/plain", "led on");
}

void handle_ledoff() {
  digitalWrite(LED_BUILTIN, HIGH);
  server.send(200, "text/plain", "led off");
}

void handle_NotFound() {
  server.send(404, "text/plain", "Not found");
}
