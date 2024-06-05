# Carbon Quantifier
Carbon Quantifier is a tool that creates a Go file with carbon impacts (kgCO2e/h) for each AWS EC2 instance type and each region that it can run in. The output can be copied into Karpenter.

## Usage

### 1. Run BoaviztAPI locally
```sh
docker run -d -p 5001:5000 ghcr.io/boavizta/boaviztapi:latest
```

### 2. Run Carbon Quantifier
```sh
# Inside 'carbon-quantifier' folder
go run carbon-quantifier.go # Takes ~10m on my machine
```

### 3. Copy into Carbon Efficient Karpenter
Copy `zz_gernerated.carbon.go` and paste it into Karpenter
