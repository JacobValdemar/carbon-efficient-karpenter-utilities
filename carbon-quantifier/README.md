# Carbon Quantifier
Carbon Quantifier is a tool that creates a Go file with carbon impacts (kgCO2e/h) for each AWS EC2 instance type and each region that it can run in. This can be copied into Karpenter.

## Usage

### 1. Run BoaviztAPI locally
```sh
git clone https://github.com/Boavizta/boaviztapi.git
cd boaviztapi
make docker-build-development
docker run -d -p 5001:5000 boavizta/boaviztapi:[TAG] # See tag in output of 'make docker-build-development'
```

### 2. Run Carbon Quantifier
```sh
# Inside 'carbon-quantifier' follder
go run carbon-quantifier.go # Takes ~12m on my machine
```

### 3. Copy into Carbon Efficient Karpenter
Copy `zz_gernerated.carbon.go` into Carbon Efficient Karpenter whereever it should be placed