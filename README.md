# jsonpatch
Go JSON Patch library, for applying RFC 6902 patches to Go objects.

There are several existing Go JSON Patch libraries, but they either operate on maps, or bytes. If another library exists to apply patches directly to struct objects, I'm not aware of it.

For Go structs, certain operations are impossible (for example, you can't add a field that doesn't exist). Thus, this library will return errors in excess of those defined by RFC 6902. Other operations are ambiguous, and have multiple valid options. This library tries to follow the Principle of Least Surprise.

Specific Behavior:
- A `remove` op on a pointer field sets it to `nil`.
- A `remove` op on a value field sets it to a default-constructed object.
- An `add` op to a struct for a field which doesn't exist returns an error.
- An `add` op to a pointer field which is nil, creates a new object.
- A `move` op to or from a struct field which doesn't exist returns an error.
- A `copy` op to or from a struct field which doesn't exist returns an error.
- A `replace` op on a struct field which doesn't exist returns an error.
- A `replace` op on a pointer field which is `nil` returns an error.

# TODO
- interfaces, where possible (e.g. replace is possible, but add is impossible)
- array types (as opposed to Slices)
- slice/array remove op
- map member move, copy op
- ops: Test
- benchmark, optimize
- get field name, if no tag exists (the same way `encoding/json` works)
- support map keys which implement encoding.TextMarshaler
