# TLock

A simple tiny centralized lock service.

Although there are some existing key lock service, like Zookeeper, Etcd, or even Redis, they don't fit my need. E.g, I want to lock multi keys at same time, not one by one. And I want to support hierachical path lock. 

**If you have some good recommendation, please tell me!**

## Key Lock

We can lock multi keys using tlock at same time, a simple example:

```
// shell1

// lock key a, b and c at same time, lock timeout is 30s
// you must do query escape in the real scenario,:-)
POST http://localhost/lock?names=a,b,c&type=key&timeout=30

// do something then unlock
DELETE http://localhost/lock?names=a,b,c&type=key

// shell2
POST http://localhost/lock?names=a,b,c&type=key&timeout=30
DELETE http://localhost/lock?names=a,b,c&type=key

```

## Path Lock

A path lock is for hierachical lock, like a file system lock. 

E.g, if we lock path "a/b/c", other can not operate its ancestor like ("a", "a/b") or descendant like ("a/b/c/d", "a/b/c/e"), but can operate its brother like ("a/b/d").

A simple example:

```
// shell1

// lock path a/b/c, a/b/d at same time, lock timeout is 30s
// you must do query escape in the real scenario,:-)
POST http://localhost/lock?names=a/b/c,a/b/d&type=path&timeout=30

// do something then unlock
DELETE http://localhost/lock?names=a/b/c,a/b/d&type=path

// shell2
POST http://localhost/lock?names=a/b/c,a/b/d&type=path&timeout=30
DELETE http://localhost/lock?names=a/b/c,a/b/d&type=path
```