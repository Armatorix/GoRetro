I'm creating a tool for a retrospective. 
As a user flow:
1. I can create a room with some settings (votes number)
2. In a room I have phases:
2.1. TICKETING (users can join and add their ticket) - for all particiapnts
2.2. BRAINSTROING (users can join the tickets) - view for all, merging for the moderators only
2.3. VOTING (users can vote on a groups) - for all participants
2.4. DISCUSSION (going through tickets one by one and making action tickets) - view for all, making action tickets for moderators only
2.5. SUMMARY (close the whole process) - view of the tickets and action tickets for all

I want to use golang with echo/v4 as backend. It should serve:
1. FE with html and tailwindcss
2. endpoints to create, list, delete rooms
3. websocket endpoint to manage all the actions in room.
