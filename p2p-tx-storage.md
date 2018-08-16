Need to design a protocol on top of DEVp2p layer, to distribute and retrieve content on public network.

Two possible approaches:
* unstructured: example flooding — resilient to churn/dynamic network
* structured: example DHT — optimized for “relatively” stable network

Will need to figure out “seeding” approach as well (i.e. how many copies are distributed)?

Content distribution protocol will need to address GDPR data location requirements (i.e. EU user data stays in EU)? _Although GDPR allows explicit permission via terms of service to store data outside EU borders_.
