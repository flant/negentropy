const { expect } = require("chai")
const {
    Worder,
    expectStatus,
    getClient,
    rootToken,
    anotherToken,
} = require("./lib")

describe.skip("Arbitrary key-value management", function () {
    const root = getClient(rootToken)
    const worder = new Worder()

    afterEach("cleanup", async function () {
        const clean = (s) => root.delete(s, expectStatus(204))
        const promises = worder.list().map(clean)
        await Promise.all(promises)
        worder.clean()
    })

    it("responds with 400 on inexistent", async () => {
        await root.get(worder.gen(), expectStatus(400))
    })

    it("puts simple data", async () => {
        const payload = { hello: "world" }
        const key = worder.gen()
        await root.post(key, payload, expectStatus(204))
    })

    it("can read created kv", async () => {
        const payload = { hello: "world" }
        const key = worder.gen()

        await root.post(key, payload, expectStatus(204))
        const { data } = await root.get(key, expectStatus(200))

        expect(data.data).to.deep.eq(payload)
    })

    it("can delete created kv", async () => {
        const payload = { hello: "world" }
        const key = worder.gen()

        await root.post(key, payload, expectStatus(204))
        const { data } = await root.get(key, expectStatus(200))

        expect(data.data).to.deep.eq(payload)

        await root.delete(key, expectStatus(204))
        await root.get(key, expectStatus(400))
    })

    it("can get a list of created keys", async () => {
        const payload = { hello: "world" }
        const keys = [worder.gen(), worder.gen()]

        await root.post(keys[0], payload, expectStatus(204))
        await root.post(keys[1], payload, expectStatus(204))
        const { data } = await root.get("?list=true", expectStatus(200))

        expect(data.data.keys).to.contain.all.members(keys)
    })

    it("can be seen by other users", async () => {
        const payload = { hello: "world" }
        const key = worder.gen()

        const other = getClient(anotherToken)

        await root.post(key, payload, expectStatus(204))
        const { data } = await other.get(key, expectStatus(200))

        expect(data.data).to.deep.eq(payload)
    })
})
