import { expect } from "chai"
import { API, EndpointBuilder, SingleFieldReponseMapper } from "./lib/api.mjs"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import {
    genGroupPayload,
    genRoleBindingPayload,
    genServiceAccountPayload,
    genTenantPayload,
    genUserPayload,
} from "./lib/payloads.mjs"

//    /tenant/{tid}/role_binding/{rbid}

describe("Role Binding", function () {
    const rootClient = getClient(rootToken)

    const rootTenantAPI = new API(
        rootClient,
        new EndpointBuilder(["tenant"]),
        new SingleFieldReponseMapper("data.tenant", "data.tenants"),
    )

    function getSubtenantClient(client, name) {
        return new API(
            client,
            new EndpointBuilder(["tenant", name]),
            new SingleFieldReponseMapper(`data.${name}`, `data.${name}s`),
        )
    }

    const rootRoleBindingClient = getSubtenantClient(rootClient, "role_binding")

    // Clients to provide dependencies
    const rootUserClient = getSubtenantClient(rootClient, "user")
    const rootServiceAccountClient = getSubtenantClient(rootClient, "service_account")
    const rootGroupClient = getSubtenantClient(rootClient, "group")

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

    async function createRoleBinding(tid, overrides) {
        const payload = genPayload(overrides)
        return await rootRoleBindingClient.create({
            params: { tenant: tid },
            payload,
        })
    }

    async function createGroup(tid) {
        const sa = await createServiceAccount(tid)

        const payload = genGroupPayload({
            subjects: [subject("service_account", sa.uuid)],
        })
        return await rootGroupClient.create({
            params: { tenant: tid },
            payload,
        })
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
        return await Promise.all([
            createUser(tid).then((x) => subject("user", x.uuid)),
            createGroup(tid).then((x) => subject("group", x.uuid)),
            createServiceAccount(tid).then((x) => subject("service_account", x.uuid)),
        ])
    }

    function subject(type, id) {
        return { type, id }
    }

    it("can be created", async () => {
        const tid = await createTenantId()
        const subjects = await createSubjects(tid)

        const rb = await rootRoleBindingClient.create({
            params: { tenant: tid },
            payload: genPayload({ subjects }),
        })

        expect(rb).to.include.keys("uuid", "tenant_uuid", "resource_version", "subjects")
        expect(rb.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(rb.tenant_uuid).to.eq(tid)
        expect(rb.resource_version).to.be.a("string").of.length.greaterThan(5)

        expect(rb.subjects).to.deep.eq(subjects)
    })

    it("can be read", async () => {
        const tenant = await createTenant()
        const tid = tenant.uuid
        const subjects = await createSubjects(tid)

        // create
        const payload = genPayload({ subjects })
        const created = await rootRoleBindingClient.create({
            params: { tenant: tid },
            payload,
        })
        const rbid = created.uuid
        const generated = {
            uuid: created.uuid,
            tenant_uuid: created.tenant_uuid,
            resource_version: created.resource_version,
        }

        // read
        const params = { tenant: tid, role_binding: rbid }
        const read = await rootRoleBindingClient.read({ params })

        const subResp = { ...payload, ...generated }
        delete subResp.ttl

        expect(read).to.deep.contain(subResp, "must contain generated fields")
        expect(read.resource_version).to.be.a("string").of.length.greaterThan(5)

        expect(read.valid_till).to.lt(Date.now() + payload.ttl)

        expect(read).to.deep.eq(
            created,
            "reading and creation responses should contain the same data",
        )
    })

    it("can be updated", async () => {
        const tid = await createTenantId()
        const subjects = await createSubjects(tid)

        // create
        const created = await createRoleBinding(tid, { subjects })

        // update
        const newSubjects = await createSubjects(tid)
        const payload = genPayload({
            resource_version: created.resource_version,
            subjects: newSubjects,
        })
        const params = { tenant: tid, role_binding: created.uuid }
        const updated = await rootRoleBindingClient.update({
            params,
            payload,
        })

        // read
        const read = await rootRoleBindingClient.read({ params })

        expect(read).to.deep.eq(updated)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()
        const subjects = await createSubjects(tid)

        // create
        const roleBinding = await createRoleBinding(tid, { subjects })
        const rbid = roleBinding.uuid

        // delete
        const params = { tenant: tid, role_binding: rbid }
        await rootRoleBindingClient.delete({ params })

        // read
        await rootRoleBindingClient.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const subjects = await createSubjects(tid)
        const roleBinding = await createRoleBinding(tid, { subjects })
        const rbid = roleBinding.uuid

        // delete
        const params = { tenant: tid }
        const list = await rootRoleBindingClient.list({ params })

        expect(list).to.be.an("array").of.length(1) // if not 1, maybe roleBindings are not filtered by tenants
        expect(list[0].uuid).to.eq(rbid)
    })

    it("can be deleted by the tenant deletion", async () => {
        const tid = await createTenantId()
        const subjects = await createSubjects(tid)
        const roleBinding = await createRoleBinding(tid, { subjects })

        await rootTenantAPI.delete({ params: { tenant: tid } })

        const params = { tenant: tid, role_binding: roleBinding.uuid }
        const opts = expectStatus(404)
        await rootRoleBindingClient.read({ params, opts })
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        const params = { role_binding: "no-such" }

        before("create tenant", async () => {
            params.tenant = await createTenantId()
        })

        it("cannot read, gets 404", async () => {
            await rootRoleBindingClient.read({ params, opts })
        })

        it("cannot update, gets 404", async () => {
            await rootRoleBindingClient.update({
                params,
                opts,
                payload: genTenantPayload(),
            })
        })

        it("cannot delete, gets 404", async () => {
            await rootRoleBindingClient.delete({ params, opts })
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
            const unauth = getSubtenantClient(client, "role_binding")
            let payload = {}

            const params = {}
            before("create tenant", async () => {
                params.tenant = await createTenantId()
            })

            beforeEach("create roleBinding payload", async () => {
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
                const rb = await rootRoleBindingClient.create({
                    params,
                    payload,
                })
                const rbid = rb.uuid

                await unauth.read({
                    params: { ...params, role_binding: rbid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const rb = await rootRoleBindingClient.create({
                    params,
                    payload,
                })
                const rbid = rb.uuid
                await unauth.update({
                    params: { ...params, role_binding: rbid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const rb = await rootRoleBindingClient.create({
                    params,
                    payload,
                })
                const rbid = rb.uuid
                await unauth.delete({
                    params: { ...params, role_binding: rbid },
                    opts,
                })
            })
        }
    })
})
