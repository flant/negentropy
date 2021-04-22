import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genTenantPayload, TenantEndpointBuilder } from "./lib/tenant.mjs"
import { expect } from "chai"
import { genUserPayload, UserEndpointBuilder } from "./lib/user.mjs"
import { API } from "./lib/api.mjs"

describe("Users", function () {
    const rootClient = getClient(rootToken)
    const rootTenantAPI = new API(rootClient, new TenantEndpointBuilder())
    const rootUserAPI = new API(rootClient, new UserEndpointBuilder())

    async function createTenantId() {
        const payload = genTenantPayload()
        const { data } = await rootTenantAPI.create({ payload })
        return data.data.id
    }

    async function createUserId(tid) {
        const payload = genUserPayload()
        const { data: body } = await rootUserAPI.create({
            params: { tenant: tid },
            payload,
        })
        return body.data.id
    }

    it("can be created", async () => {
        const tid = await createTenantId()

        const { data: body } = await rootUserAPI.create({
            params: { tenant: tid },
            payload: genUserPayload(),
        })

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.have.key("id")
        expect(body.data.id).to.be.a("string").of.length.above(10)
    })

    it("can be read", async () => {
        const tid = await createTenantId()

        // create
        const payload = genUserPayload()
        const { data: body } = await rootUserAPI.create({
            params: { tenant: tid },
            payload,
        })
        const id = body.data.id

        // read
        const { data: user } = await rootUserAPI.read({
            params: { tenant: tid, user: id },
        })
        expect(user.data).to.deep.eq(payload)
    })

    it("responds with 404 on inexisting", async () => {
        const tid = await createTenantId()

        await rootUserAPI.read({
            params: { tenant: tid, user: "no-such" },
            opts: { validateStatus: (s) => s === 404 },
        })
    })

    it("can be updated", async () => {
        const tid = await createTenantId()

        // create
        const uid = await createUserId(tid)

        // update
        const payload = genUserPayload()
        const params = { tenant: tid, user: uid }
        await rootUserAPI.update({ params, payload })

        // read
        const { data: body } = await rootUserAPI.read({ params })
        const user = body.data

        expect(user).to.deep.eq(payload)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const uid = await createUserId(tid)

        // delete
        const params = { tenant: tid, user: uid }
        await rootUserAPI.delete({ params })

        // read
        await rootUserAPI.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const uid = await createUserId(tid)

        // delete
        const params = { tenant: tid }
        const { data: body } = await rootUserAPI.list({ params })

        expect(body.data).to.be.an("object").and.include.keys("ids")
        expect(body.data.ids).to.be.an("array").of.length(1)
        expect(body.data.ids[0]).to.eq(uid)
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        const params = { user: "no-such" }

        before("create tenant", async () => {
            params.tenant = await createTenantId()
        })

        it("cannot read, gets 404", async () => {
            await rootUserAPI.read({ params, opts })
        })

        it("cannot update, gets 404", async () => {
            await rootUserAPI.update({
                params,
                opts,
                payload: genTenantPayload(),
            })
        })

        it("cannot delete, gets 404", async () => {
            await rootUserAPI.delete({ params, opts })
        })
    })

    describe("access", function () {
        describe("when unauthenticated", function () {
            runWithClient(getClient(), 400)
        })

        describe("when unauthorized", function () {
            runWithClient(getClient("xxx"), 403)
        })

        function runWithClient(client, expectedStatus) {
            const opts = expectStatus(expectedStatus)
            const unauth = new API(client, new UserEndpointBuilder())
            let payload = {}

            const params = {}
            before("create tenant", async () => {
                params.tenant = await createTenantId()
            })

            beforeEach("create user payload", async () => {
                payload = genUserPayload()
            })

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ params, payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ params, opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const { data } = await rootUserAPI.create({
                    params,
                    payload,
                })
                const uid = data.data.id

                await unauth.read({
                    params: { ...params, user: uid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const { data } = await rootUserAPI.create({ params, payload })
                const uid = data.data.id
                await unauth.update({
                    params: { ...params, user: uid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const { data } = await rootUserAPI.create({ params, payload })
                const uid = data.data.id
                await unauth.delete({
                    params: { ...params, user: uid },
                    opts,
                })
            })
        }
    })
})
