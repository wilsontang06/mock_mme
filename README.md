# Mock MME

Mock MME is a Diameter S6A client for the purpose of acting as a testing environment for HSS's, 
pertaining to 4G LTE celluar networks.


This project currently has implementations for load testing ULR's to an HSS as well as testing 
similar ULR's to multiple HSS's (for a distributed HSS).


The HSS used while building this testing framework was UW ICTD Lab's CoLTE project.


The settings that are in the implementation right now relies on these facts about the HSS:
* There are two HSS's running on the local machine on localhost
* One HSS is running on port 3868 and the second is running on port 3869
* oai_hss only allows support for one mme (mme.OpenAir5G.Alliance)?


## Build

To run the code without building it:
(You must have Golang set up to run this command)
```
go run *.go
```

To build the code into a binary executable:
```
go build *.go
```

Or if you want to just run the binary executable already provided:
```
./mock_mme
```
