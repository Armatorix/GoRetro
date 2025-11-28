// English translations for GoRetro
window.translations = window.translations || {};

window.translations.en = {
    // Navigation
    nav: {
        title: "GoRetro",
        user: "User"
    },
    
    // Index page
    index: {
        pageTitle: "GoRetro - Retrospective Tool",
        createRoom: {
            title: "Create New Retrospective",
            roomNameLabel: "Room Name",
            roomNamePlaceholder: "Sprint Retrospective",
            votesLabel: "Votes per User",
            createButton: "Create Room"
        },
        myRooms: {
            title: "My Retrospectives",
            noRooms: "No retrospectives yet. Create one to get started!",
            ownerBadge: "Owner"
        },
        howItWorks: {
            title: "How It Works",
            step1: {
                title: "Ticketing",
                description: "Participants add their thoughts and feedback"
            },
            step2: {
                title: "Brainstorming",
                description: "Moderators group similar tickets together"
            },
            step3: {
                title: "Voting",
                description: "Everyone votes on what to discuss"
            },
            step4: {
                title: "Discussion",
                description: "Discuss top items and create action tickets"
            },
            step5: {
                title: "Summary",
                description: "Review all feedback and action items"
            }
        }
    },
    
    // Room page
    room: {
        pageTitle: "{roomName} - GoRetro",
        shareLink: "Share link:",
        connectionStatus: {
            connecting: "Connecting...",
            connected: "Connected",
            disconnected: "Disconnected",
            error: "Error",
            reconnecting: "Reconnecting... ({attempt}/{max})",
            failed: "Connection failed"
        },
        pendingApproval: {
            title: "Waiting for Approval",
            message: "Your request to join this room is pending approval by the moderator or owner.",
            pleaseWait: "Please wait while your access is being reviewed."
        },
        phases: {
            ticketing: "1. Ticketing",
            brainstorming: "2. Brainstorming",
            voting: "3. Voting",
            discussion: "4. Discussion",
            summary: "5. Summary"
        },
        votes: {
            info: "Votes used: {used} / {total}"
        },
        tickets: {
            title: "Tickets",
            addButton: "+ Add Ticket",
            placeholder: "What's on your mind?",
            cancel: "Cancel",
            submit: "Submit",
            noTickets: "No tickets yet. Be the first to add one!",
            votes: "{count} votes",
            coveredBadge: "✓ Covered",
            markCovered: "Mark as covered/discussed",
            markNotCovered: "Mark as not covered",
            unmergeAll: "Unmerge all",
            separateFromParent: "Separate from parent",
            deleteConfirmTitle: "Delete Ticket",
            deleteConfirmMessage: "Are you sure you want to delete this ticket? This action cannot be undone.",
            delete: "Delete"
        },
        actions: {
            title: "Action Items",
            addButton: "+ Add Action",
            placeholder: "Describe the action item...",
            assignLabel: "Assign to (click to select/deselect multiple):",
            selectAll: "Select All",
            deselectAll: "Deselect All",
            assignedTo: "Assigned to: {names}",
            noActions: "No action items yet.",
            cancel: "Cancel",
            add: "Add Action",
            deleteConfirm: "Are you sure you want to delete this action item?"
        },
        participants: {
            title: "Participants",
            owner: "owner",
            moderator: "moderator",
            participant: "participant",
            pending: "pending",
            pendingApprovalTitle: "Pending Approval",
            promote: "↑ Promote",
            demote: "↓ Demote",
            remove: "✗ Remove",
            approve: "✓ Approve",
            reject: "✗ Reject",
            promoteToModerator: "Promote to moderator",
            demoteFromModerator: "Demote from moderator",
            removeFromRoom: "Remove from room",
            removeConfirm: "Are you sure you want to remove this participant from the room?",
            rejectConfirm: "Are you sure you want to reject this participant?",
            noParticipants: "No participants"
        },
        messages: {
            cannotPerformDisconnected: "Cannot perform action: disconnected from server",
            unableToReconnect: "Unable to reconnect. Please refresh the page.",
            accessApproved: "Your access has been approved!",
            requestRejected: "Your request to join was rejected"
        }
    },
    
    // Common
    common: {
        cancel: "Cancel",
        submit: "Submit",
        delete: "Delete",
        close: "Close",
        save: "Save"
    }
};
