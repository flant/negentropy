import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genTenantPayload, TenantEndpointBuilder } from "./lib/tenant.mjs"
import { expect } from "chai"
import {
    genRoleBindingPayload,
    SubTenantEntrypointBuilder,
} from "./lib/subtenant.mjs"
import { API } from "./lib/api.mjs"
import { genGroupPayload } from "./lib/subtenant.mjs"
import { genUserPayload } from "./lib/subtenant.mjs"
import { genServiceAccountPayload } from "./lib/subtenant.mjs"

//    /tenant/{tid}/role_binding/{rbid}

describe("Role Binding", function () {
    const rootClient = getClient(rootToken)
    const rootTenantAPI = new API(rootClient, new TenantEndpointBuilder())

    // Rolebinding API access
    const roleBindingEntrypointBuilder = new SubTenantEntrypointBuilder(
        "role_binding",
    )
    const rootRoleBindingClient = new API(
        rootClient,
        roleBindingEntrypointBuilder,
    )

    // Clients to provide dependencies
    const userEntrypointBuilder = new SubTenantEntrypointBuilder("user")
    const groupEntrypointBuilder = new SubTenantEntrypointBuilder("group")
    const saEntrypointBuilder = new SubTenantEntrypointBuilder(
        "service_account",
    )
    const rootUserClient = new API(rootClient, userEntrypointBuilder)
    const rootGroupClient = new API(rootClient, groupEntrypointBuilder)
    const rootServiceAccountClient = new API(rootClient, saEntrypointBuilder)

    function genPayload(override) {
        return genRoleBindingPayload(override)
    }

    async function createTenant() {
        const payload = genTenantPayload()
        const { data } = await rootTenantAPI.create({ payload })
        return data.data
    }

    async function createTenantId() {
        const tenant = await createTenant()
        return tenant.uuid
    }

    async function createRoleBinding(tid, overrides) {
        const payload = genPayload(overrides)
        const { data: body } = await rootRoleBindingClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data
    }

    async function createGroup(tid) {
        const sa = await createServiceAccount(tid)

        const payload = genGroupPayload({
            subjects: [subject("service_account", sa.uuid)],
        })
        const { data: body } = await rootGroupClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data
    }

    async function createServiceAccount(tid) {
        const payload = genServiceAccountPayload()
        const { data: body } = await rootServiceAccountClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data
    }

    async function createUser(tid) {
        const payload = genUserPayload()
        const { data: body } = await rootUserClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data
    }

    async function createSubjects(tid) {
        const user = await createUser(tid)
        const group = await createGroup(tid)
        const sa = await createServiceAccount(tid)
        return [
            subject("group", group.uuid),
            subject("user", user.uuid),
            subject("service_account", sa.uuid),
        ]
    }

    function subject(type, id) {
        return { type, id }
    }

    it("can be created", async () => {
        const tid = await createTenantId()
        const subjects = await createSubjects(tid)

        const { data: body } = await rootRoleBindingClient.create({
            params: { tenant: tid },
            payload: genPayload({ subjects }),
        })

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.include.keys(
            "uuid",
            "tenant_uuid",
            "resource_version",
            "subjects",
        )
        expect(body.data.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(body.data.tenant_uuid).to.eq(tid)
        expect(body.data.resource_version)
            .to.be.a("string")
            .of.length.greaterThan(5)

        expect(body.data.subjects).to.deep.eq(subjects)
    })

    it("can be read", async () => {
        const tenant = await createTenant()
        const tid = tenant.uuid
        const subjects = await createSubjects(tid)

        // create
        const payload = genPayload({ subjects })
        const { data: created } = await rootRoleBindingClient.create({
            params: { tenant: tid },
            payload,
        })
        const rbid = created.data.uuid
        const generated = {
            uuid: created.data.uuid,
            tenant_uuid: created.data.tenant_uuid,
            resource_version: created.data.resource_version,
        }

        // read
        const params = { tenant: tid, role_binding: rbid }
        const { data: read } = await rootRoleBindingClient.read({ params })

        const subResp = { ...payload, ...generated }
        delete subResp.ttl

        expect(read.data).to.deep.contain(
            subResp,
            "must contain generated fields",
        )
        expect(read.data.resource_version)
            .to.be.a("string")
            .of.length.greaterThan(5)

        expect(read.data.valid_till).to.lt(Date.now() + payload.ttl)

        expect(read.data).to.deep.eq(
            created.data,
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
        const { data: updated } = await rootRoleBindingClient.update({
            params,
            payload,
        })

        // read
        const { data: read } = await rootRoleBindingClient.read({ params })

        expect(read.data).to.deep.eq(updated.data)
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
        const { data: body } = await rootRoleBindingClient.list({ params })

        expect(body.data).to.be.an("object").and.include.keys("uuids")
        expect(body.data.uuids).to.be.an("array").of.length(1) // if not 1, maybe roleBindings are not filtered by tenants
        expect(body.data.uuids[0]).to.eq(rbid)
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
            const unauth = new API(client, roleBindingEntrypointBuilder)
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
                const { data } = await rootRoleBindingClient.create({
                    params,
                    payload,
                })
                const rbid = data.data.uuid

                await unauth.read({
                    params: { ...params, role_binding: rbid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const { data } = await rootRoleBindingClient.create({
                    params,
                    payload,
                })
                const rbid = data.data.uuid
                await unauth.update({
                    params: { ...params, role_binding: rbid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const { data } = await rootRoleBindingClient.create({
                    params,
                    payload,
                })
                const rbid = data.data.uuid
                await unauth.delete({
                    params: { ...params, role_binding: rbid },
                    opts,
                })
            })
        }
    })
})
