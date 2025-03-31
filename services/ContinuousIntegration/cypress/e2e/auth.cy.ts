describe("Full Spec", { testIsolation: false }, () => {
  const newUsername = "invitee";
  const newPassword = "secure123";
  var signupUrl = "";
  before(() => {
    Cypress.session.clearAllSavedSessions();
    cy.task("db:seed", "default");
    cy.clearAllCookies();
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
  it("Admin can send an invitation", () => {
    cy.task("db:seed", "default");
    // Login as admin/existing user
    cy.login("tester");

    // Visit the invite page
    cy.visit("/invite");
    cy.url().should("include", "/invite");

    // Fill out and submit the invite form
    cy.get("#username").type(newUsername);
    cy.get("button[type='submit']").click();

    // Check if invitation was successful
    cy.contains("Invitation Created Successfully").should("be.visible");

    // Get the invitation URL and save it for next test
    cy.get("input[readonly]")
      .invoke("val")
      .then((url: string) => {
        signupUrl = url;
      });
  });

  it("New user can accept invitation", () => {
    // Visit the stored invitation URL
    cy.visit(signupUrl);

    // Check that we're on the accept invite page
    cy.contains("Invitation to Join").should("be.visible");
    cy.contains(newUsername).should("be.visible");

    // Fill password and accept invitation
    cy.get("#password").type(newPassword);
    cy.get("button[type='submit']").click();

    // Should be redirected to dashboard after successful acceptance
    cy.url().should("equal", "http://localhost:8080/");
  });

  it("New user can access organization resources", () => {
    // The user should already be logged in from accepting the invite
    // Verify they can access organization repos
    cy.visit("/");
    cy.contains("Repos").should("be.visible");

    // Should not be redirected to login
    cy.url().should("not.include", "/login");
  });

  it("New user can log out and log back in", () => {
    // First log out
    cy.visit("/login");
    Cypress.session.clearAllSavedSessions();

    // Now log back in with new credentials
    cy.visit("/login");
    cy.get("#username").type(newUsername);
    cy.get("#password").type(newPassword);
    cy.get("#login").click();

    // Should be logged in successfully
    cy.url().should("equal", "http://localhost:8080/");
    cy.contains("Repos").should("be.visible");
  });

  it("Cannot send invite for existing username", () => {
    // Login as admin again
    Cypress.session.clearAllSavedSessions();
    cy.login("tester");

    // Try to invite user with same username
    cy.visit("/invite");
    cy.get("#username").type(newUsername);
    cy.get("button[type='submit']").click();

    // Should be redirected back to the form without success message
    cy.contains("Invitation Created Successfully").should("not.exist");
  });
});
