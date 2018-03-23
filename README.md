# Trust Net Blockchain
A public DLT based implementation for Bring Your Own Identity (BYOI) system.

## What is Trust Net?
What are high level objectives of trust network?
* Build your own Identity service
* Privacy (You control who sees what about your Identity)
* Security (No single point of mass vulnerability)
* Availability (Resilient to high number of node failures)
* Ownership (You decide where/how to host your identity)

## Why Trust Net?
Fundamental question:
> there are already identity services, blockchain networks, ... how is this proposal different from whats out there?

Before we answer this question, lets re-cap what a blockchain really offers:
* a mechanism to “formulate an agreement” between “un-trusting” peers, without relying on any one centralized and trusted authority
* a mechanism to “transfer value” between “un-trusting” peers, without relying on any one centralized and trusted mediator
* a mechanism to control “access" to information by authorized party, without relying on any one centralized and trusted access service

With above 3 key value offerings of blockchain networks, lets evaluate existing solutions against our offering:
* any Identity solution out there is a SaaS based model, which means a single centralized entity owns the data/protocol/API/framework and eventually the access/usage of the identities. This means, you can not swap one identity service provider with another in such an Identity as a Service model. Also, there is no decentralized access to information — which may be fine, but raises concerns about privacy, security and long term availability (service cost)
* our solution is about building a distributed Identity network, which means there is no one service or entity that controls the access/usage of the Identity. An individual person owns his/her identity, and they have complete control over privacy of their identity information  (i.e. who can see what specific information about their identity). Identity is distributed across the blockchain network, and when the identity owner permits access to a specific attribute of their identity by a specific entity on the network, then that entity can access that information from any “node” in the network
* our solution will allow buulding services in the network that may offer “proof of identity” to certify some specific attributes of a person’s Identity. However that “proof” would still be owned by the individual and they'll have control on who can access that information across the network

## How Trust Net Works?
All this is good, but then how will applications get RBAC capability on these blockchain identities?
* we’ll pass RBAC onus to application itself, and provide a generic consensus platform that can be used to build a "shard" over the main identity network that provide "roles" to individual identity (and potentially audit access)
* our value proposition is to build true “ownership”, “privacy” and “security” into digital Identities

How will an application authenticate a user’s identity on blockchain?
* will use decryption challenge using asymmetric public key of the user that is known/published on the blockchain
* this also implies, challenger needs to have similar identity on the blockchain, since response would be encrypted using challenger’s public key

What are the major components?
* Client protocol
* Identity protocol
* Reputation/Trust protocol
* Confidence protocol
* Endorsement protocol
* Consensus protocol
* Reference Node Service
* Reference Identity Service
* Reference Certifying Agency Service
* Reference Transaction Service
* Reference Consensus Service
* Reference Client
