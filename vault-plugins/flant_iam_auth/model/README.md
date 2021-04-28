After adding new entity model, add processing into:

[kafka_source-Self](../io/kafka_source/self.go) - `Restore()` func and `processMessage()` func - for `flant_iam_auth` models

[kafka_source-Root](../io/kafka_source/root.go) - `Restore()` func and `processMessage()` func - for `flant_iam` models


[kafka_destination-Self](../io/kafka_destination/self.go) - `isValidObjectType()` func

