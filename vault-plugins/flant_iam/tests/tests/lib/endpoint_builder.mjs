export class EndpointBuilder {
    constructor(fields = []) {
        this.fields = fields
    }

    one(p = {}, q = {}) {
        const parts = this.concat(this.fields, p, true)
        return "/" + parts.join("/") + stringifyQuery(q)
    }

    collection(p = {}, q = {}) {
        const parts = this.concat(this.fields, p, false)
        return "/" + parts.join("/") + stringifyQuery(q)
    }

    privileged(p = {}, q = {}) {
        const lastField = this.fields[this.fields.length - 1]
        p[lastField] = "privileged"
        const parts = this.concat(this.fields, p, true)
        return "/" + parts.join("/") + stringifyQuery(q)
    }

    concat(fields, values, demandTail = false) {
        const parts = []

        for (let i = 0; i < fields.length; i++) {
            const field = fields[i]
            parts.push(field)

            const value = values[field]

            const isLast = i === fields.length - 1
            const notLast = !isLast

            if (demandTail || notLast) {
                if (!value) {
                    const valstr = JSON.stringify(values)
                    const msg = `expected to have value for field "${field}", got ${valstr}`
                    throw new Error(msg)
                }
            }

            if (isLast) {
                if (demandTail) {
                    parts.push(value)
                }
                break
            }

            parts.push(value)
        }
        return parts
    }
}

export function stringifyQuery(q = {}) {
    if (!q || Object.keys(q).length === 0) {
        return ""
    }
    return "?" + new URLSearchParams(q).toString()
}

// Example class, not to be used
export class ExampleEndpointBuilder {
    constructor() {
        this.prefix = "items"
    }

    one(p = {}, q = {}) {
        return join(this.prefix, p.item) + stringifyQuery(q)
    }

    collection(p = {}, q = {}) {
        return this.prefix + stringifyQuery(q)
    }

    privileged(p = {}, q = {}) {
        return join(this.prefix, "privileged") + stringifyQuery(q)
    }
}
