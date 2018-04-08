# TrustNet Blockchain
A public Bring Your Own Identity (BYOI) system with _"DLT stack as a library"_.

## Why TrustNet?
Fundamental question:
> _"Why do we need yet another blockchain network?"_

Before we answer this question, lets re-cap what current blockchain networks offer:
* Anonymous private identities
* Promiscuous public transactions
* Applications as "second-class" citizen
* Single ledger network as the applications controller

Above properties make it very difficult, if not impossible, to build **decentralized native applications** that require following:
* a strong and well known identity of the user
* application level privacy and encryption of transactions
* DLT support in native applications
* multi-network applications

This is the reason TrustNet was created and designed from ground up _"DLT stack as a library"_, making it possible to build DLT based native applications.

## What TrustNet Does?
**For Users:**
* Bring Your Own Identity (BYOI)
* Privacy (You control who sees what about your Identity)
* Security (No single point of mass vulnerability)
* Availability (Resilient to high number of node failures)
* Ownership (You decide which node on the network can be used as an Identity service!)

**For Applications:**
* A public Identity Management Network
* Private applications using strong public identities

## How TrustNet Works?
* Provides application driven DLT library
* Abstracts protocol and consensus into layers
* Manages “world state” based on consensus

```
Application (transaction business logic)
     /\
     ||
     \/
Consensus Platform (protocol layer)
     /\
     ||
     \/
DAG Consensus Engine (consensus layer)
```

## DLT Comparison
|Feature|TrustNet|Ethereum|
|----|----|----|
|Objective|Identity and privacy|Smart contracts|
|Identity Model|Strong Public identity, BYOI Identity management system|Anonymous private identities|
|Privacy Model|Strong privacy, application level encryption|Public transactions|
|Application Model|Application is the driver|Application is a plugin|
|DLT Stack Model|Multi-network, stack as a library|single network, stack as the controller|
