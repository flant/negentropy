This module working with root kafka queue. Root queue is queue synchronize entities from flant_iam to iam_auth

# Files

`message_dispatcher` - handle raw kafka message, cast message to model object and call object handler

`object_handler` - implementation of object handler called from `message_dispatcher`

`restore` - handler that called when restore objects from queue when plugin start 