This module working with self kafka queue. Self queue is persistent store for iam_auth models.

# Files

`message_dispatcher` - handle raw kafka message, cast message to model object and call object handler

`object_handler` - implementation of object handler called from `message_dispatcher`

`restore` - handler that called when restore objects from queue when plugin start 