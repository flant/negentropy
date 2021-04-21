const {expect} = require("chai")
const {Worder, expectStatus, getClient, rootToken, anotherToken} = require("./lib")

const TEN = "tenant"

describe("Tenants", function () {
    const root = getClient(rootToken)
    const worder = new Worder()

    // afterEach("cleanup", async function () {
    //     const clean = s => root.delete(`tenant/${s}`, expectStatus(204))
    //     const promises = worder.list().map(clean)
    //     await Promise.all(promises)
    //     worder.clean()
    // })


    it("cannot be created without name", async () => {
        await root.post(TEN, {}, {validateStatus: s => s >= 400})
    })


    it("cannot be created with empty name", async () => {
        await root.post(TEN, {name: ""}, expectStatus(400))
    })

    it("can be created with a name", async () => {
        const payload = {name: worder.gen()}
        const {data} = await root.post(TEN, payload, expectStatus(204))
    })

    it("can be listed", async () => {
        const payload = {name: worder.gen()}
        await root.post(TEN, payload, expectStatus(204))

        const {data} = await root.get("tenant?list=true", expectStatus(200))

        expect(data.data).to.be("array").and.to.be.not.empty
    })

    it("have identifying fields in list", async () => {
        const payload = {name: worder.gen()}
        await root.post(TEN, payload, expectStatus(204))

        const {data: body} = await root.get("tenant?list=true", expectStatus(200))

        expect(body.data).to.be("array").and.to.be.not.empty
        const tenant = body.data[0]
        expect(tenant).to.include.keys("id", "name")
        expect(tenant.name).to.eq(payload.name)
    })
})

