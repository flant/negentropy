import { expect } from "chai"
import { v4 as uuidv4 } from "uuid"
import { API, EndpointBuilder, SingleFieldReponseMapper } from "./lib/api.mjs"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genTenantPayload } from "./lib/payloads.mjs"

describe("Tenant", function () {
    const rootClient = getClient(rootToken)

    function getAPIClient(client) {
        return new API(
            client,
            new EndpointBuilder(["tenant"]),
            new SingleFieldReponseMapper("data.tenant", "data.uuids"),
        )
    }

    const root = getAPIClient(rootClient)

    describe("payload", () => {
        describe("identifier", () => {
            const invalidCases = [
                {
                    title: "number allowed", // the matter of fact ¯\_(ツ)_/¯
                    payload: genTenantPayload({ identifier: 100 }),
                    validateStatus: (x) => x === 201,
                },
                {
                    title: "absent identifier forbidden",
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
                    await root.create({
                        payload: x.payload,
                        opts: {
                            validateStatus: x.validateStatus,
                        },
                    })
                }),
            )
        })
    })

    it("can be created", async () => {
        const payload = genTenantPayload()

        const tenant = await root.create({ payload })

        expect(tenant).to.include.keys("uuid", "identifier", "resource_version")
        expect(tenant.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(tenant.resource_version).to.be.a("string").of.length.greaterThan(5)
    })

    it("can be read", async () => {
        const payload = genTenantPayload()

        const tenCreated = await root.create({ payload })
        const tenRead = await root.read({ params: { tenant: tenCreated.uuid } })

        expect(tenRead).to.deep.eq(tenCreated)
        expect(tenCreated).to.deep.contain({
            ...payload,
            uuid: tenCreated.uuid,
        })
        expect(tenCreated.resource_version).to.be.a("string").of.length.greaterThan(5)
    })

    it("can be read by id", async () => {
        const payload1 = genTenantPayload()
        const payload2 = genTenantPayload()
        const payload3 = genTenantPayload()

        const tenCreated1 = await root.create({ payload: payload1 })
        const tenCreated2 = await root.create({ payload: payload2 })
        const tenCreated3 = await root.create({ payload: payload3 })

        const id1 = tenCreated1.uuid
        const id2 = tenCreated2.uuid
        const id3 = tenCreated3.uuid

        const idParam = (id) => ({ params: { tenant: id } })
        const tenRead1 = await root.read(idParam(id1))
        const tenRead2 = await root.read(idParam(id2))
        const tenRead3 = await root.read(idParam(id3))

        expect(tenRead1).to.contain({ ...payload1, uuid: id1 })
        expect(tenRead2).to.contain({ ...payload2, uuid: id2 })
        expect(tenRead3).to.contain({ ...payload3, uuid: id3 })
    })

    it("can be updated", async () => {
        const createPld = genTenantPayload()
        const updatePld = genTenantPayload()

        // create
        const tenCreated1 = await root.create({ payload: createPld })
        const uuid = tenCreated1.uuid
        const resource_version = tenCreated1.resource_version

        // update
        await root.update({
            params: {
                tenant: uuid,
            },
            payload: {
                ...updatePld,
                resource_version,
            },
        })

        // read
        const tenant = await root.read({ params: { tenant: uuid } })

        expect(tenant).to.contain({ ...updatePld, uuid }, "payload must be saved")
        expect(tenant.resource_version)
            .to.be.a("string")
            .of.length.greaterThan(5)
            .and.not.to.eq(resource_version, "resource version must be updated")
    })

    it("can be deleted", async () => {
        const payload = genTenantPayload()

        const tenCreated1 = await root.create({ payload })
        const id = tenCreated1.uuid

        const params = { tenant: id }
        await root.delete({ params })

        await root.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        const payload = genTenantPayload()
        const t = await root.create({ payload })

        const list = await root.list()

        expect(list).to.be.an("array").and.to.contain(t.uuid)
    })

    it("has identifying fields in list", async () => {
        const payload = genTenantPayload()
        const t = await root.create({ payload })
        const id = t.uuid

        const list = await root.list()

        expect(list).to.include(id)
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        it("cannot read, gets 404", async () => {
            await root.read({ params: { tenant: "no-such" }, opts })
        })

        it("cannot update, gets 404", async () => {
            await root.update({ params: { tenant: "no-such" }, payload: genTenantPayload(), opts })
        })

        it("cannot delete, gets 404", async () => {
            await root.delete({ params: { tenant: "no-such" }, opts })
        })
    })

    describe("no access", function () {
        describe("when unauthenticated", function () {
            runWithClient(getClient(), 400)
        })

        describe("when unauthorized", function () {
            runWithClient(getClient("xxx"), 403)
        })

        function runWithClient(client, expectedStatus) {
            const unauth = getAPIClient(client)
            const opts = expectStatus(expectedStatus)

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ payload: genTenantPayload(), opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const t = await root.create({ payload: genTenantPayload() })
                await unauth.read({ params: { tenant: t.uuid }, opts })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const t = await root.create({ payload: genTenantPayload() })
                const params = { tenant: t.uuid }
                await unauth.update({ params, payload: genTenantPayload(), opts })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const t = await root.create({ payload: genTenantPayload() })
                await unauth.delete({ params: { tenant: t.uuid }, opts })
            })
        }
    })

    describe("privileged access", function () {
        it(`creates`, async () => {
            const payload = genTenantPayload({ uuid: uuidv4() })

            const t = await root.createPrivileged({ payload })

            expect(t.uuid).to.deep.eq(payload.uuid)
        })
    })
})
