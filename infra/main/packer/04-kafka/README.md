# Testing commands.
Firstly source environment variables:
```bash
source /etc/kafka/scripts/variables.sh
```

## Create a topic.
```bash
/opt/kafka/bin/kafka-topics.sh --create --topic test-1 --command-config /tmp/kafka/client-ssl.properties -bootstrap-server $FQDN:9092
```

## List topics.
```bash
/opt/kafka/bin/kafka-topics.sh --list --command-config /tmp/kafka/client-ssl.properties -bootstrap-server $FQDN:9092
```

## Produce messages to the topic.
This will open terminal where you will be able to type any strings and produce them by typing return.
```bash
/opt/kafka/bin/kafka-console-producer.sh --topic test-1 --producer.config /tmp/kafka/client-ssl.properties --bootstrap-server $FQDN:9092
```

## Consume messages from the topic.
This will open terminal and print all messages from the beginning of the topic. Also, all new messages will be printed.
```bash
/opt/kafka/bin/kafka-console-consumer.sh --topic test-1 --consumer.config /tmp/kafka/client-ssl.properties --from-beginning --bootstrap-server $FQDN:9092
```
> ℹ️ For some reason exiting from the above two commands will log out you from the console.
