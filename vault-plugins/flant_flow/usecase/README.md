# Usecase module

## Goal

This module describes business logic and ties inputs from external sources to model layer that contains provides DB
access.

```
backend -> Usecase  -> Repository -> Memdb

```

## Conventions:

All usecases contain transaction since it is a kind of a context for all operations. It is either stored in a field of a
usecase struct of it is the first argument in usecase function.
