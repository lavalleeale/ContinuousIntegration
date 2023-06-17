describe("Full Spec", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.task("db:seed", "default");
  });
  it("Logs In", () => {
    cy.login("tester");
  });
  it("Creates Repo", () => {
    cy.login("tester");
    cy.visit("/");
    cy.get("#show-add-repo").click();
    cy.get("#url").type("https://github.com/sshca/sshca", { force: true });
    cy.get("#add-repo").click({ force: true });
  });
  it("Creates Build", () => {
    cy.login("tester");
    cy.visit("/");
    cy.contains("sshca").click();
    cy.get("#show-add-build").click();
    cy.fixture("basic.json").then((sshca) => {
      cy.get("#command").type(JSON.stringify(sshca), {
        force: true,
        parseSpecialCharSequences: false,
        delay: 0,
      });
      cy.get("#add-build").click({ force: true });
    });
  });
  it("Runs Build", () => {
    cy.login("tester");
    cy.visit("/");
    cy.contains("sshca").click();
    cy.wait(1000);
    cy.get("a > .paper").click();
    cy.get(".bg-yellow-500").should("be.visible");
    cy.get(".bg-gray-500").should("be.visible");
    cy.get(".bg-green-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-yellow-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-green-500", { timeout: 1000000 }).should("have.length", 2);
    cy.contains("go").click();
    cy.contains("/neededFiles/dev/docker-compose.yaml").should("be.visible");
  });

  it("Deletes Repo", () => {
    cy.login("tester");
    cy.visit("/");
    cy.get(".fa-trash").click();
    cy.contains("sshca").should("not.exist");
  });
});
