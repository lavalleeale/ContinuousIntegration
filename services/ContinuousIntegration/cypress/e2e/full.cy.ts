describe("Full Spec", { testIsolation: false }, () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.task("db:seed", "default");
  });
  it("Logs In", () => {
    cy.visit("/");
    cy.login("tester");
  });
  it("Creates Repo", () => {
    cy.get("#show-add-repo").click();
    cy.get("#url").type("http://10.0.1.18:3000/alex/sshca", { force: true });
    cy.get("#add-repo").click({ force: true });
  });
  it("Creates Build", () => {
    cy.get("#show-add-build").click();
    cy.fixture("sshca.json").then((sshca) => {
      cy.get("#command").type(JSON.stringify(sshca), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
      cy.get("#add-build").click({ force: true });
    });
  });
  it("Runs Build", () => {
    cy.get(".bg-yellow-500").should("be.visible");
    cy.get(".bg-gray-500").should("be.visible");
    cy.contains("build").click();
    cy.contains(
      "The postinstall script automatically ran `prisma generate` and did not find your `prisma/schema.prisma`."
    ).should("not.exist");
    cy.contains(
      "The postinstall script automatically ran `prisma generate` and did not find your `prisma/schema.prisma`.",
      { timeout: 1000000 }
    ).should("exist");
    cy.go("back");
    cy.get(".bg-green-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-yellow-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-green-500", { timeout: 1000000 }).should("have.length", 2);
  });
});
