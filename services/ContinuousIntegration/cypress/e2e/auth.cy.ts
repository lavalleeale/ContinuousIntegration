describe("Full Spec", { testIsolation: false }, () => {
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.task("db:seed", "default");
  });
  it("Fails to log in with unknown user", () => {
    cy.visit("/");
    cy.url().should("include", "/login");
    cy.get("#username").type("unknown");
    cy.get("#password").type("unknown");
    cy.get("#login").click();
    cy.url().should("include", "/login");
    cy.contains("user not found").should("be.visible");
  });
  it("Fails to log in with wrong password", () => {
    cy.visit("/login");
    cy.get("#username").type("tester");
    cy.get("#password").type("unknown");
    cy.get("#login").click();
    cy.url().should("include", "/login");
    cy.contains("user not found").should("be.visible");
  });
  it("Fails to access api with no session", () => {
    ["addRepoGithub", "", "repo/1", "build/1", "build/1/container/1"].forEach(
      (path) => {
        cy.visit(`/${path}`);
        cy.url().should("include", "/login");
      }
    );
  });
  it("Accesses pages with wrong user", () => {
    cy.task("db:seed", "invalidAccess");
    cy.login("user2");
    ["repo/1", "build/1", "build/1/container/1"].forEach((path) => {
      cy.visit(`/${path}`);
      cy.url().should("equal", "http://localhost:8080/");
    });
  });
});
