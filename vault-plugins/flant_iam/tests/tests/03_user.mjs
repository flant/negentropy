import { expect } from "chai"
import { API, EndpointBuilder, SingleFieldReponseMapper } from "./lib/api.mjs"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genMultipassPayload, genTenantPayload, genUserPayload } from "./lib/payloads.mjs"

//    /tenant/{tid}/user/{uid}

describe("User", function () {
    const rootClient = getClient(rootToken)

    const rootTenantAPI = new API(
        rootClient,
        new EndpointBuilder(["tenant"]),
        new SingleFieldReponseMapper("data.tenant", "data.tenants"),
    )

    function getAPIClient(client) {
        return new API(
            client,
            new EndpointBuilder(["tenant", "user"]),
            new SingleFieldReponseMapper("data.user", "data.users"),
        )
    }

    const rootUserAPI = getAPIClient(rootClient)

    function genPayload(override) {
        return genUserPayload(override)
    }

    async function createTenant() {
        const payload = genTenantPayload()
        return await rootTenantAPI.create({ payload })
    }

    async function createTenantId() {
        const tenant = await createTenant()
        return tenant.uuid
    }

    async function createUser(tid) {
        const payload = genPayload()
        return await rootUserAPI.create({
            params: { tenant: tid },
            payload,
        })
    }

    it("can be created", async () => {
        const tid = await createTenantId()

        const user = await rootUserAPI.create({
            params: { tenant: tid },
            payload: genPayload(),
        })

        expect(user).to.include.keys("uuid", "tenant_uuid", "resource_version")
        expect(user.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(user.tenant_uuid).to.eq(tid)
        expect(user.resource_version).to.be.a("string").of.length.greaterThan(5)
    })

    it("can be read", async () => {
        const tenant = await createTenant()
        const tid = tenant.uuid
        // create
        const payload = genPayload()
        const created = await rootUserAPI.create({
            params: { tenant: tid },
            payload,
        })
        const uid = created.uuid
        const generated = {
            email: "",
            uuid: created.uuid,
            tenant_uuid: created.tenant_uuid,
            resource_version: created.resource_version,
            full_identifier: payload.identifier + "@" + tenant.identifier,
            origin: "iam",
            extensions: null,
        }

        // read
        const read = await rootUserAPI.read({
            params: { tenant: tid, user: uid },
        })

        expect(read).to.deep.eq({ ...payload, ...generated }, "must have generated fields")
        expect(read).to.deep.eq(
            created,
            "reading and creation responses should contain the same data",
        )
        expect(read.resource_version).to.be.a("string").of.length.greaterThan(5)
    })

    it("can be updated", async () => {
        const tid = await createTenantId()

        // create
        const created = await createUser(tid)

        // update
        const payload = genPayload({
            resource_version: created.resource_version,
        })
        const params = { tenant: tid, user: created.uuid }
        const updated = await rootUserAPI.update({
            params,
            payload,
        })

        // read
        const read = await rootUserAPI.read({ params })
        expect(read).to.deep.eq(updated)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const user = await createUser(tid)
        const uid = user.uuid

        // delete
        const params = { tenant: tid, user: uid }
        await rootUserAPI.delete({ params })

        // read
        await rootUserAPI.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const user = await createUser(tid)
        const uid = user.uuid

        // delete
        const params = { tenant: tid }
        const list = await rootUserAPI.list({ params })

        expect(list).to.be.an("array").of.length(1) // if not 1, maybe users are not filtered by tenants
        expect(list[0].uuid).to.eq(uid)
    })

    it("can be deleted by the tenant deletion", async () => {
        const tid = await createTenantId()
        const user = await createUser(tid)

        await rootTenantAPI.delete({ params: { tenant: tid } })

        const params = { tenant: tid, user: user.uuid }
        const opts = expectStatus(404)
        await rootUserAPI.read({ params, opts })
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
            const unauth = getAPIClient(client)
            let payload = {}

            const params = {}
            before("create tenant", async () => {
                params.tenant = await createTenantId()
            })

            beforeEach("create user payload", async () => {
                payload = genPayload()
            })

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ params, payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ params, opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const user = await rootUserAPI.create({
                    params,
                    payload,
                })
                const uid = user.uuid

                await unauth.read({
                    params: { ...params, user: uid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const user = await rootUserAPI.create({
                    params,
                    payload,
                })
                const uid = user.uuid
                await unauth.update({
                    params: { ...params, user: uid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const user = await rootUserAPI.create({
                    params,
                    payload,
                })
                const uid = user.uuid
                await unauth.delete({
                    params: { ...params, user: uid },
                    opts,
                })
            })
        }
    })

    describe("multipass", function () {
        const endpointBuilder = new EndpointBuilder(["tenant", "user", "multipass"])
        const rootMPClient = new API(
            rootClient,
            endpointBuilder,
            new SingleFieldReponseMapper("data.multipass", "data.multipasses"),
        )

        async function createMultipass(t, u, override = {}) {
            const payload = genMultipassPayload(override)
            const params = { tenant: t, user: u }
            const mp = await rootMPClient.create({ params, payload })
            return mp
        }

        it("can be created", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)

            await createMultipass(t.uuid, u.uuid)
        })

        it("contains expected data when created", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)

            const mp = await createMultipass(t.uuid, u.uuid)

            expect(mp)
                .to.be.an("object")
                .and.include.keys(
                    "allowed_cidrs",
                    "allowed_roles",
                    "description",
                    "max_ttl",
                    "owner_type",
                    "owner_uuid",
                    "tenant_uuid",
                    "origin",
                    "extensions",
                    "ttl",
                    "uuid",
                    "valid_till",
                )
                .and.not.include.keys("salt")

            expect(mp.uuid, "uuid").to.be.a("string")
            expect(mp.owner_type, "owner_type").to.be.a("string")
            expect(mp.owner_uuid, "owner_uuid").to.be.a("string")
            expect(mp.tenant_uuid, "tenant_uuid").to.be.a("string")

            expect(mp.salt, "salt").to.be.undefined

            expect(mp.description, "description").to.be.a("string")

            expect(mp.allowed_cidrs, "allowed_cidrs").to.be.an("array")
            expect(mp.allowed_roles, "allowed_roles").to.be.an("array")

            expect(mp.ttl, "ttl").to.be.a("number")
            expect(mp.max_ttl, "max_ttl").to.be.a("number")
            expect(mp.valid_till, "valid_till")
                .to.be.a("number")
                .greaterThan(Date.now() / 1e3)
        })

        it("can be read", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)
            const created = await createMultipass(t.uuid, u.uuid)

            const params = {
                tenant: t.uuid,
                user: u.uuid,
                multipass: created.uuid,
            }
            const read = await rootMPClient.read({ params })

            expect(read).to.deep.eq(created)
        })

        it("can be listed", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)

            const createId = () => createMultipass(t.uuid, u.uuid).then((mp) => mp.uuid)

            const ids = await Promise.all([createId(), createId(), createId()])

            const params = {
                tenant: t.uuid,
                user: u.uuid,
            }

            const list = await rootMPClient.list({ params })

            expect(list.map((mp) => mp.uuid)).to.have.all.members(ids)
            for (const mp of list) {
                expect(mp.salt).to.be.undefined
            }
        })

        it("can be deleted", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)
            const created = await createMultipass(t.uuid, u.uuid)

            const params = {
                tenant: t.uuid,
                user: u.uuid,
                multipass: created.uuid,
            }

            await rootMPClient.delete({ params })

            await rootMPClient.read({ params, opts: expectStatus(404) })
        })

        it("cannot be updated", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)
            const createdMP = await createMultipass(t.uuid, u.uuid)

            const params = {
                tenant: t.uuid,
                user: u.uuid,
                multipass: createdMP.uuid,
            }

            await rootMPClient.update({
                params,
                payload: genMultipassPayload(),
                opts: expectStatus(405),
            })
        })
    })
})
