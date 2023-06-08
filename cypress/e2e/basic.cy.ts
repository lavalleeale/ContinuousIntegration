const login = (name: string) => {
  cy.session("Login", () => {
    cy.visit("http://localhost:8080");
    cy.url().should("contain", "/login");
    cy.get("#username").type(name);
    cy.get("#password").type(name);
    cy.get("#login").click();
    cy.url().should("eq", "http://localhost:8080/");
  });
};

describe("Basic Spec", () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.task("db:seed", "default");
  });
  it("Logs In", () => {
    login("tester");
  });
  it("Creates Repo", () => {
    login("tester");
    cy.visit("/");
    cy.get("#show-add-repo").click();
    cy.get("#url").type("https://github.com/sshca/sshca", { force: true });
    cy.get("#add-repo").click({ force: true });
  });
  it("Creates Build", () => {
    login("tester");
    cy.visit("/");
    cy.contains("sshca").click();
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
    login("tester");
    cy.visit("/");
    cy.contains("sshca").click();
    cy.wait(1000);
    cy.get("a > .paper").click();
    cy.get(".bg-yellow-500").should("be.visible");
    cy.get(".bg-gray-500").should("be.visible");
    cy.get(".bg-green-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-yellow-500", { timeout: 1000000 }).should("be.visible");
    cy.get(".bg-green-500", { timeout: 1000000 }).should("have.length", 2);
  });
});
