import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genTenantPayload, TenantAPI } from "./lib/tenant.mjs"
import { expect } from "chai"
import { v4 as uuidv4 } from "uuid"

describe("Tenants", function() {
    const rootClient = getClient(rootToken)
    const root = new TenantAPI(rootClient)

    // afterEach("cleanup", async function () {
    //     const clean = s => root.delete(`tenant/${s}`, expectStatus(204))
    //     const promises = worder.list().map(clean)
    //     await Promise.all(promises)
    //     worder.clean()
    // })

    describe("payload", () => {
        describe("identifier", () => {
            const invalidCases = [
                {
                    title: "number allowed",
                    payload: genTenantPayload({ identifier: 100 }),
                    validateStatus: (x) => x === 201,
                },
                {
                    title: "absent name field forbidden",
                    payload: (() => {
                        const p = genTenantPayload({})
                        delete p.identifier
                        return p
                    })(),
                    validateStatus: (x) => x === 400,
                },
                {
                    title: "empty string forbidden",
                    payload: genTenantPayload({ identifier: "" }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "array forbidden",
                    payload: genTenantPayload({ identifier: ["a"] }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "object forbidden",
                    payload: genTenantPayload({ identifier: { a: 1 } }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
            ]

            invalidCases.forEach((x) =>
                it(x.title, async () => {
                    await root.create(x.payload, {
                        validateStatus: x.validateStatus,
                    })
                }),
            )
        })
    })

    it("can be created", async () => {
        const payload = genTenantPayload()

        const { data: body } = await root.create(payload)

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.have.key("id")
        expect(body.data.id).to.be.a("string").of.length.above(10)
    })

    it("can be read", async () => {
        const payload = genTenantPayload()

        const { data: body } = await root.create(payload)
        const id = body.data.id

        const { data: tenant } = await root.read(id)
        expect(tenant.data).to.deep.eq({ ...payload, id })
    })

    it("can be read by id", async () => {
        const payload1 = genTenantPayload()
        const payload2 = genTenantPayload()
        const payload3 = genTenantPayload()

        const { data: body1 } = await root.create(payload1)
        const id1 = body1.data.id
        const { data: body2 } = await root.create(payload2)
        const id2 = body2.data.id
        const { data: body3 } = await root.create(payload3)
        const id3 = body3.data.id

        const { data: resp1 } = await root.read(id1)
        const { data: resp2 } = await root.read(id2)
        const { data: resp3 } = await root.read(id3)

        expect(resp1.data).to.deep.eq({ ...payload1, id: id1 })
        expect(resp2.data).to.deep.eq({ ...payload2, id: id2 })
        expect(resp3.data).to.deep.eq({ ...payload3, id: id3 })
    })

    it("can be updated", async () => {
        const createPld = genTenantPayload()
        const updatePld = genTenantPayload()

        // create
        const { data: body1 } = await root.create(createPld)
        const id = body1.data.id

        // update
        const { data: body2 } = await root.update(id, updatePld)

        // read
        const { data: body3 } = await root.read(id)
        const tenant = body3.data

        expect(tenant).to.deep.eq({ ...updatePld, id })
    })

    it("can be deleted", async () => {
        const createPld = genTenantPayload()

        const { data: body1 } = await root.create(createPld)
        const id = body1.data.id

        await root.delete(id)

        await root.read(id, expectStatus(404))
    })

    it("can be listed", async () => {
        const payload = genTenantPayload()
        await root.create(payload)

        const { data } = await root.list()

        expect(data.data).to.be.an("object")
    })

    it("has identifying fields in list", async () => {
        const payload = genTenantPayload()
        const { data: creationBody } = await root.create(payload)
        const id = creationBody.data.id

        const { data: listBody } = await root.list()

        expect(listBody.data).to.be.an("object").and.have.key("ids")
        const { ids } = listBody.data
        expect(ids).to.include(id)
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        it("cannot read, gets 404", async () => {
            await root.read("no-such", opts)
        })

        it("cannot update, gets 404", async () => {
            await root.update("no-such", genTenantPayload(), opts)
        })

        it("cannot delete, gets 404", async () => {
            await root.delete("no-such", opts)
        })
    })

    describe("no access", function() {
        describe("when unauthenticated", function() {
            runWithClient(getClient(), 400)
        })

        describe("when unauthorized", function() {
            runWithClient(getClient("xxx"), 403)
        })

        function runWithClient(client, expectedStatus) {
            const unauth = new TenantAPI(client)
            const opts = expectStatus(expectedStatus)

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create(genTenantPayload(), opts)
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list(opts)
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const { data } = await root.create(genTenantPayload())
                await unauth.read(data.data.id, opts)
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const { data } = await root.create(genTenantPayload())
                await unauth.update(data.data.id, genTenantPayload(), opts)
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const { data } = await root.create(genTenantPayload())
                await unauth.delete(data.data.id, opts)
            })
        }
    })

    describe("privileged access", function() {
        it(`creates`, async () => {
            const p = genTenantPayload()
            p.id = uuidv4()

            const { data: body } = await root.createPriveleged(p)

            const id = body.data.id
            expect(id).to.deep.eq(p.id)
        })
    })
})
