import { expect } from "chai"
import { API, EndpointBuilder, SingleFieldReponseMapper } from "./lib/api.mjs"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import {
    genGroupPayload,
    genServiceAccountPayload,
    genTenantPayload,
    genUserPayload,
} from "./lib/payloads.mjs"

//    /tenant/{tid}/group/{gid}

describe("Group", function () {
    const rootClient = getClient(rootToken)

    const rootTenantAPI = new API(
        rootClient,
        new EndpointBuilder(["tenant"]),
        new SingleFieldReponseMapper("data.tenant", "data.uuids"),
    )

    function getSubtenantClient(client, name) {
        return new API(
            client,
            new EndpointBuilder(["tenant", name]),
            new SingleFieldReponseMapper("data." + name, "data.uuids"),
        )
    }

    const rootGroupClient = getSubtenantClient(rootClient, "group")

    // Clients to provide dependencies
    const rootUserClient = getSubtenantClient(rootClient, "user")
    const rootServiceAccountClient = getSubtenantClient(rootClient, "service_account")

    function genPayload(override) {
        return genRoleBindingPayload(override)
    }

    async function createTenant() {
        const payload = genTenantPayload()
        return await rootTenantAPI.create({ payload })
    }

    async function createTenantId() {
        const tenant = await createTenant()
        return tenant.uuid
    }

    async function createServiceAccount(tid) {
        const payload = genServiceAccountPayload()
        return await rootServiceAccountClient.create({
            params: { tenant: tid },
            payload,
        })
    }

    async function createUser(tid) {
        const payload = genUserPayload()
        return await rootUserClient.create({
            params: { tenant: tid },
            payload,
        })
    }

    async function createSubjects(tid) {
        const user = await createUser(tid)
        const sa = await createServiceAccount(tid)
        return [subject("user", user.uuid), subject("service_account", sa.uuid)]
    }

    function subject(type, id) {
        return { type, id }
    }

    function genPayload(override) {
        return genGroupPayload(override)
    }

    async function createGroup(tid) {
        const subjects = await createSubjects(tid)
        const payload = genPayload({ subjects })
        return await rootGroupClient.create({
            params: { tenant: tid },
            payload,
        })
    }

    it("can be created", async () => {
        const tid = await createTenantId()
        const subjects = await createSubjects(tid)

        const group = await rootGroupClient.create({
            params: { tenant: tid },
            payload: genPayload({ subjects }),
        })

        expect(group).to.include.keys("uuid", "tenant_uuid", "resource_version")
        expect(group.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(group.tenant_uuid).to.eq(tid)
        expect(group.resource_version).to.be.a("string").of.length.greaterThan(5)

        expect(group.subjects).to.deep.eq(subjects)
    })

    it("can be read", async () => {
        const tenant = await createTenant()
        const tid = tenant.uuid
        const subjects = await createSubjects(tid)

        // create
        const payload = genPayload({ subjects })
        const created = await rootGroupClient.create({
            params: { tenant: tid },
            payload,
        })
        const gid = created.uuid
        const generated = {
            uuid: created.uuid,
            tenant_uuid: created.tenant_uuid,
            resource_version: created.resource_version,
            full_identifier: payload.identifier + "@group." + tenant.identifier,
            origin: "iam",
            extensions: null,
        }

        // read
        const read = await rootGroupClient.read({
            params: { tenant: tid, group: gid },
        })

        expect(read).to.deep.eq({ ...payload, ...generated }, "must have generated fields")
        expect(read).to.deep.eq(
            created,
            "reading and creation responses should contain the same data",
        )
        expect(read.resource_version).to.be.a("string").of.length.greaterThan(5)

        expect(read.subjects).to.deep.eq(subjects)
    })

    it("can be updated", async () => {
        const tid = await createTenantId()

        // create
        const created = await createGroup(tid)

        // update
        const subjects = await createSubjects(tid)
        const payload = genPayload({
            resource_version: created.resource_version,
            subjects,
        })
        const params = { tenant: tid, group: created.uuid }
        const updated = await rootGroupClient.update({
            params,
            payload,
        })

        // read
        const read = await rootGroupClient.read({ params })

        expect(read).to.deep.eq(updated)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const group = await createGroup(tid)
        const gid = group.uuid

        // delete
        const params = { tenant: tid, group: gid }
        await rootGroupClient.delete({ params })

        // read
        await rootGroupClient.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const group = await createGroup(tid)
        const gid = group.uuid

        // delete
        const params = { tenant: tid }
        const list = await rootGroupClient.list({ params })

        expect(list).to.be.an("array").of.length(1) // if not 1, maybe groups are not filtered by tenants
        expect(list[0]).to.eq(gid)
    })

    it("can be deleted by the tenant deletion", async () => {
        const tid = await createTenantId()
        const group = await createGroup(tid)

        await rootTenantAPI.delete({ params: { tenant: tid } })

        const params = { tenant: tid, group: group.uuid }
        const opts = expectStatus(404)
        await rootGroupClient.read({ params, opts })
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        const params = { group: "no-such" }

        before("create tenant", async () => {
            params.tenant = await createTenantId()
        })

        it("cannot read, gets 404", async () => {
            await rootGroupClient.read({ params, opts })
        })

        it("cannot update, gets 404", async () => {
            await rootGroupClient.update({
                params,
                opts,
                payload: genTenantPayload(),
            })
        })

        it("cannot delete, gets 404", async () => {
            await rootGroupClient.delete({ params, opts })
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
            const unauth = getSubtenantClient(client, "group")
            let payload = {}

            const params = {}
            before("create tenant", async () => {
                params.tenant = await createTenantId()
            })

            beforeEach("create group payload", async () => {
                const subjects = await createSubjects(params.tenant)
                payload = genPayload({ subjects })
            })

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ params, payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ params, opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const group = await rootGroupClient.create({
                    params,
                    payload,
                })
                const gid = group.uuid

                await unauth.read({
                    params: { ...params, group: gid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const group = await rootGroupClient.create({
                    params,
                    payload,
                })
                const gid = group.uuid
                await unauth.update({
                    params: { ...params, group: gid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const group = await rootGroupClient.create({
                    params,
                    payload,
                })
                const gid = group.uuid
                await unauth.delete({
                    params: { ...params, group: gid },
                    opts,
                })
            })
        }
    })
})
