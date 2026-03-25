# viola-platform

Viola Technologies — MVP monorepo for an AI-native XDR platform.

## What runs (MVP)
- `agent` produces endpoint telemetry to Kafka
- `ingestion` normalizes telemetry
- `detection` emits detection hits + alerts
- `workers` aggregates alerts into incidents

## Quick start (local)
```bash
make dev        # start Postgres + Kafka + ClickHouse + Redis
make protos     # generate Go protos from shared/proto

# in separate terminals
cd services/ingestion && go run ./cmd/ingestion
cd services/detection && go run ./cmd/detection
cd services/workers && go run ./cmd/workers
cd agent && go run ./cmd/viola-agent
```

## Non-negotiables
- `tenant_id` everywhere.
- `request_id` everywhere.
- Schema versioning: breaking changes => new topic version.




 We've completed both major deliverables. Here are your options for what to
   build next:

  ---
  Option 1: Expand Viola Labs Platform 🚀

  Continue building the convergence company infrastructure:

  A. AI Engine Implementation

  - Build molecular property prediction models (PyTorch)
  - Implement active learning for experiment design
  - Create generative chemistry models
  - Add model serving infrastructure

  B. Robotics Controller

  - Implement ROS 2 integration
  - Add robot motion planning (MoveIt 2)
  - Create simulation environment (Gazebo)
  - Build fleet management system

  C. Frontend Dashboard

  - Build React web interface
  - Add real-time experiment monitoring
  - Create data visualization (Plotly charts)
  - Implement user management

  D. Full Docker Deployment

  - Create Docker Compose for local development
  - Add Kubernetes manifests for production
  - Set up CI/CD pipelines
  - Configure monitoring stack

  ---
  Option 2: Deploy Gateway API to Production 🔐

  Take the existing Gateway API to production:

  A. Production Infrastructure

  - Set up Kubernetes cluster
  - Deploy PostgreSQL with replication
  - Configure Kafka cluster
  - Add load balancers

  B. Monitoring & Observability

  - Set up Prometheus + Grafana
  - Configure alerting (PagerDuty)
  - Add distributed tracing (Jaeger)
  - Create dashboards

  C. Documentation

  - Write API documentation
  - Create runbooks for operations
  - Document incident response procedures
  - Build developer guides

  ---
  Option 3: Build a Different Convergence Company 💡

  Apply the convergence blueprint to a different domain:

  A. Alternative Convergence Wedges

  - AI + Robotics + Energy (smart grid optimization)
  - AI + Sensors + Healthcare (continuous health monitoring)
  - AI + Drones + Agriculture (precision farming)
  - AI + Robotics + Manufacturing (generative design + 3D printing)

  B. Start from Strategic Planning

  - Choose new convergence wedge
  - Design technical architecture
  - Build integration layer
  - Create business blueprint

  ---
  Option 4: Deep Dive on Specific Component 🔬

  Focus on one specific technology in depth:

  A. Advanced AI/ML

  - Build production ML pipeline
  - Implement MLOps best practices
  - Create AutoML system
  - Add federated learning

  B. Robotics Deep Dive

  - Build complex manipulation tasks
  - Implement reinforcement learning for control
  - Create digital twins
  - Add computer vision pipelines

  C. Security & Compliance

  - Implement zero-trust architecture
  - Add compliance frameworks (SOC 2, ISO 27001)
  - Build audit system
  - Create security testing suite

  ---
  Option 5: Something Completely Different 🎯

  Tell me what you'd like to build and I'll architect and implement it.

  ---
  Please choose an option (1-5) or tell me specifically what you'd like me 
  to proceed with.

  I'm ready to:
  - Write more code
  - Design new systems
  - Build complete features
  - Create documentation
  - Deploy infrastructure
  - Or anything else you have in mind

  What would you like?