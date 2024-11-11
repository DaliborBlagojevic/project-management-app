# Koristimo zvaničnu Go sliku kao osnovu
FROM golang:1.22.9 as build_container

# Postavljamo radni direktorijum
WORKDIR /app

# Kopiramo go.mod i go.sum kako bi preuzeli zavisnosti
COPY go.mod ./
COPY go.sum ./

# Preuzimamo Go module
RUN go mod download

# Kopiramo sav kod uključujući proto fajlove
COPY . .

# Ovdje ne radimo build jer `proto` verovatno sadrži samo .proto fajlove za generisanje koda
# Pretpostavljam da koristiš neki alat za generisanje Go kodova iz `.proto` fajlova (kao što je `protoc`)

# Instaliramo protoc (ako nije već instaliran u bazične Go slike)
RUN apt-get update && apt-get install -y protobuf-compiler

# Generišemo Go kodove iz .proto fajlova
RUN protoc --go_out=. --go-grpc_out=. ./proto/*.proto

# Onda ćemo pretpostaviti da će se koristi generisani Go kod u ostalim mikroservisima

# Za finalnu sliku, ne moraš ništa dalje da kopiraš, samo možeš ostaviti generisane fajlove u /app
# Ovdje je opcionalno, jer možeš koristiti samo build container za izgradnju finalnih servisa
