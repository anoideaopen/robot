##Non-functional requirements
* [ ] number of tokens - 8 channels are already default for current HLF implementation, we are planning 6 ABTs at launch and at least 40 ITs shortly after
* [ ] share grafana dashboards with Us
* [x] Configurable log verbosity determined by environment vars
* [X] Private keys and credentials should be stored in vault. They should not be committed to source and should not leak to the browser/client (ex: chrome dev tools source navigation)
* [X] PII, or other personal information should never be logged
* [X] Credentials, and authn jwt tokens are not logged in the pod stdout/logs
* [x] All services are fault-tolerant and can recover if they crash
* [x] All services are stateless and load-balanced to enable rolling deployments without downtime
* [ ] All services are replicated and do not have a single point of failure (including infra services like postgresql and mongodb - should have at least 2 pods - also cert-manager and prometheus-operator)
* [X] All services have a fast startup, no more than 20sec, and if a health check fails, it is taken out of the load balancer. Also fast failure, if a microservice module crashes, it stops the whole microservice within 5sec
* [ ] Microservices are equipped with unit + integration tests: modules, components, services, controllers, etc, that run automatically during build → 70% code coverage
* [ ] Existing load and performance testing are documented with instructions how to set-up, configure and run (ideally, we receive some help from NT to walk us through this). This includes JMeter GRPC load testing and end-to-end load tests with Overload.yandex.ru  Stretch: Load tests + E2E tests are automated via CI/CD pipelines
* [ ] All Microservices are equipped with sonarqube scripts to capture code coverage and output to a report
* [x] All microservices README are updated with context about the microservice, its business function, and what services it talks to
* [ ] All third party libs must have either no license or MIT/BSD. CopyLeft licenses like GNU should not be used.
* [ ] Configuration determined with environment vars to be able to deploy each service to any US environment
* [x] All micro-service releases are tagged and CI/CD pipeline allows to redeploy from a release tag
* [x] HLF Validator Network: No certificates/keys committed to source (including non production environments) - all PKI material is stored in Vault only.
* [x] HLF test-network exposes telemetry (either via jaegar and/or grafana metrics, in the same way micro-services expose metrics for both prod and non-prod environments)
* [ ] HLF-Client library: For production, evaluate a service discovery mechanism (more details) to replace peer configurations in source (alternative: write them to vault) -- otherwise storing all validator/peer PKI in source is cumbersome and a security risk
* [ ] The Platform supports a maximum of 50 channels.
* [X] The Platform should offer developers tools that provide inter-channel interaction, or, in other words, interaction between issued tokens.

