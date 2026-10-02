STACK_NAME=dev-colibri-sdk-go

start:
	docker-compose -p ${STACK_NAME} up -d --remove-orphans

stop:
	docker-compose -p ${STACK_NAME} stop

restart: stop start

clean:
	docker-compose -p ${STACK_NAME} down -v

logs:
	docker-compose -p ${STACK_NAME} logs -f

# Prometheus and Grafana only, with the dashboards and alert rules of observability/ mounted.
observability:
	docker-compose -p ${STACK_NAME} up -d prometheus grafana
