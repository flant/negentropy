
class Error(Exception):
    """ Base class for all errors. """
    pass


class InvalidMigrationError(Error):
    """ Migration contains an error. """
    pass


class InvalidNameError(Error):
    """ Migration has an invalid filename. """

    def __init__(self, filename):
        msg = 'Migration filenames must start with a UTC timestamp. ' \
              'The following file has an invalid name: %s' % filename
        super(InvalidNameError, self).__init__(msg)
