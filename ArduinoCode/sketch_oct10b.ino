#define laserModule 2   
#define laserSensor 3      
#define relay 6            
#define redLed 7
#include <Servo.h>
Servo linearActuator;

bool prosesAktif = false;  

void setup() {
  Serial.begin(9600);

  pinMode(laserSensor, INPUT);
  pinMode(relay, OUTPUT);
  pinMode(laserModule, OUTPUT);
  pinMode(LED_BUILTIN, OUTPUT);
  pinMode(redLed, OUTPUT);

  linearActuator.attach(8);

  digitalWrite(LED_BUILTIN, LOW);
  digitalWrite(laserModule, HIGH);
  digitalWrite(relay, HIGH);  
  digitalWrite(redLed, LOW);
}

void loop() {
 

  if (Serial.available() > 0) {
    String data = Serial.readStringUntil('\n'); 
    data.trim();

    if (data == "606") {
      digitalWrite(relay, LOW);   
      digitalWrite(LED_BUILTIN, HIGH);

      delay(300);

      for (int i = 0; i < 3; i++) {
        linearActuator.write(360);   
        delay(1500);
      }

      prosesAktif = true; 
    }
  }

  if (prosesAktif) {
    int sensorValue = digitalRead(laserSensor);
    int ledValue = digitalRead(redLed);
    // Serial.println(sensorValue);
    if (sensorValue == 0 && ledValue == 0) {
      digitalWrite(redLed, HIGH);

      for (int i = 0; i < 3; i++) {
        linearActuator.write(0);    
        delay(1500);
      }

      digitalWrite(relay, HIGH);      
      prosesAktif = false;
    }else if(sensorValue == 1 && ledValue == 1){
      digitalWrite(redLed, LOW);

      for (int i = 0; i < 3; i++) {
        linearActuator.write(0);    
        delay(1500);
      }

      digitalWrite(relay, HIGH);      
      prosesAktif = false;
    }
  }
}
