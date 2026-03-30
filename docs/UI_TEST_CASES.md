# UI Test Cases

## Prerequisites

Before executing any test case (unless otherwise noted in Preconditions):

- **Browser**: Modern Chromium-based browser or Safari (PWA support required)
- **URL**: https://littleliver.fly.dev/
- **Account**: A Google account available for OAuth login
- **Baby profile**: At least one baby profile created (for tests in categories 3-8)
- **Test data**: Some metric entries already logged (for dashboard, trends, and report tests)
- **Second account**: A second Google account (for invite/join tests)
- **Network**: Stable internet connection (unless testing offline/error scenarios)

---

## 1. Authentication & First-Time User Flow

### TC-01-01: Login page renders correctly
- **Preconditions**: User is not logged in
- **Steps**:
  1. Navigate to https://littleliver.fly.dev/login
  2. Observe the page content
- **Expected Result**: Page shows "LittleLiver" heading, "Post-Kasai baby health tracking" text, and a "Sign in with Google" link
- **Requires Human**: No

### TC-01-02: Sign in with Google
- **Preconditions**: User is not logged in
- **Steps**:
  1. Navigate to /login
  2. Click "Sign in with Google"
  3. Complete Google OAuth flow
- **Expected Result**: User is redirected to / (home). If user has no babies, the FirstLogin view is shown. If user has babies, the TodayDashboard is shown.
- **Requires Human**: Yes (Google OAuth)

### TC-01-03: Unauthenticated user sees login prompt on home page
- **Preconditions**: User is not logged in
- **Steps**:
  1. Navigate to /
  2. Wait for loading to complete
- **Expected Result**: Page shows "LittleLiver" heading, "Post-Kasai baby health tracking" text, and a "Sign in to get started" link pointing to /login
- **Requires Human**: No

### TC-01-04: First login — choice screen renders
- **Preconditions**: User is logged in, has no baby profiles
- **Steps**:
  1. Navigate to /
- **Expected Result**: "Welcome to LittleLiver" heading appears. Two buttons visible: "Create a Baby" and "Join with Invite Code". Description text: "Get started by creating a baby profile or joining an existing one."
- **Requires Human**: Yes (Google login required)

### TC-01-05: First login — navigate to Create Baby form
- **Preconditions**: User is logged in, has no baby profiles, on FirstLogin choice screen
- **Steps**:
  1. Click "Create a Baby"
- **Expected Result**: CreateBabyForm appears with fields: Name, Date of birth, Sex, Diagnosis date, Kasai date. A "Back" button is visible to return to the choice screen.
- **Requires Human**: Yes

### TC-01-06: First login — navigate to Join with Invite Code form
- **Preconditions**: User is logged in, has no baby profiles, on FirstLogin choice screen
- **Steps**:
  1. Click "Join with Invite Code"
- **Expected Result**: JoinBabyForm appears with an "Invite code" text input and "Join" button. A "Back" button is visible to return to the choice screen.
- **Requires Human**: Yes

### TC-01-07: First login — back button from Create Baby
- **Preconditions**: User is on the Create Baby form (from FirstLogin)
- **Steps**:
  1. Click "Back"
- **Expected Result**: Returns to the choice screen with "Create a Baby" and "Join with Invite Code" buttons
- **Requires Human**: Yes

### TC-01-08: First login — back button from Join form
- **Preconditions**: User is on the Join with Invite Code form (from FirstLogin)
- **Steps**:
  1. Click "Back"
- **Expected Result**: Returns to the choice screen with "Create a Baby" and "Join with Invite Code" buttons
- **Requires Human**: Yes

### TC-01-09: Logout
- **Preconditions**: User is logged in
- **Steps**:
  1. Click "Logout" button in the navigation header
- **Expected Result**: Button text changes to "Logging out..." while processing. User is redirected to /login after logout completes.
- **Requires Human**: Yes

### TC-01-10: NavHeader hidden when not logged in
- **Preconditions**: User is not logged in
- **Steps**:
  1. Navigate to /
  2. Observe the page header area
- **Expected Result**: No navigation header is rendered (NavHeader only renders content inside the `{#if $currentUser}` block)
- **Requires Human**: No

---

## 2. Baby Management

### TC-02-01: Create baby — all required fields
- **Preconditions**: User is logged in, on the Create Baby form
- **Steps**:
  1. Enter "Test Baby" in Name field
  2. Select a date of birth
  3. Select "Male" for Sex
  4. Click "Create Baby"
- **Expected Result**: Button shows "Creating..." while submitting. On success, user is taken to the TodayDashboard with baby name displayed.
- **Requires Human**: Yes

### TC-02-02: Create baby — with optional fields
- **Preconditions**: User is logged in, on the Create Baby form
- **Steps**:
  1. Enter "Test Baby" in Name field
  2. Select a date of birth
  3. Select "Female" for Sex
  4. Enter a Diagnosis date
  5. Enter a Kasai date
  6. Click "Create Baby"
- **Expected Result**: Baby is created successfully with all fields saved
- **Requires Human**: Yes

### TC-02-03: Create baby — validation: missing name
- **Preconditions**: User is logged in, on the Create Baby form
- **Steps**:
  1. Leave Name empty
  2. Select a date of birth and sex
  3. Click "Create Baby"
- **Expected Result**: Validation message "Name is required" appears (role="alert"). Form is not submitted.
- **Requires Human**: Yes

### TC-02-04: Create baby — validation: missing date of birth
- **Preconditions**: User is logged in, on the Create Baby form
- **Steps**:
  1. Enter a name
  2. Leave Date of birth empty
  3. Select a sex
  4. Click "Create Baby"
- **Expected Result**: Validation message "Date of birth is required" appears. Form is not submitted.
- **Requires Human**: Yes

### TC-02-05: Create baby — validation: missing sex
- **Preconditions**: User is logged in, on the Create Baby form
- **Steps**:
  1. Enter a name and date of birth
  2. Leave Sex as "Select..."
  3. Click "Create Baby"
- **Expected Result**: Validation message "Sex is required" appears. Form is not submitted.
- **Requires Human**: Yes

### TC-02-06: Create baby — server error
- **Preconditions**: User is logged in, on the Create Baby form, API returns an error
- **Steps**:
  1. Fill in all required fields
  2. Click "Create Baby" (with backend returning an error)
- **Expected Result**: Error message "Failed to create baby" is displayed
- **Requires Human**: Yes

### TC-02-07: Switch between babies (multiple babies)
- **Preconditions**: User has 2+ baby profiles linked
- **Steps**:
  1. Navigate to /settings
  2. In the "Active Baby" section, observe the BabySelector dropdown
  3. Select a different baby from the dropdown
- **Expected Result**: Dashboard and all views update to reflect the newly selected baby. The dropdown shows all babies as options.
- **Requires Human**: Yes

### TC-02-08: Baby selector — single baby shows name only
- **Preconditions**: User has exactly 1 baby profile
- **Steps**:
  1. Observe the BabySelector in the NavHeader or Settings
- **Expected Result**: Baby name is displayed as plain text (not a dropdown), since there is only one baby
- **Requires Human**: Yes

### TC-02-09: Edit baby settings
- **Preconditions**: User is logged in, has a baby, on /settings
- **Steps**:
  1. Modify the Name field in Baby Settings
  2. Change the Kasai date
  3. Click "Save Settings"
- **Expected Result**: Button shows "Saving..." during submission. Settings are saved successfully. Form retains updated values.
- **Requires Human**: Yes

### TC-02-10: Edit baby settings — change default cal per feed with recalculate
- **Preconditions**: User is logged in, has a baby with existing feeding entries, on /settings
- **Steps**:
  1. Change "Default cal per feed" to a new value
  2. Observe that a "Recalculate existing feeding calories" checkbox appears
  3. Check the recalculate checkbox
  4. Click "Save Settings"
- **Expected Result**: Settings are saved and existing feeding calorie values are recalculated based on the new default
- **Requires Human**: Yes

### TC-02-11: Edit baby settings — change default cal per feed without recalculate
- **Preconditions**: User is logged in, has a baby, on /settings
- **Steps**:
  1. Change "Default cal per feed" to a new value
  2. Leave the "Recalculate existing feeding calories" checkbox unchecked
  3. Click "Save Settings"
- **Expected Result**: Settings are saved. Existing feeding calorie values remain unchanged. Only new feedings use the updated default.
- **Requires Human**: Yes

### TC-02-12: Edit baby settings — validation: empty name
- **Preconditions**: User is logged in, on /settings
- **Steps**:
  1. Clear the Name field
  2. Click "Save Settings"
- **Expected Result**: Validation message "Name is required" appears. Form is not submitted.
- **Requires Human**: Yes

### TC-02-13: Generate invite code
- **Preconditions**: User is logged in, has a baby, on /settings
- **Steps**:
  1. Scroll to the "Invite Code" section
  2. Click "Generate Invite Code"
- **Expected Result**: Button shows "Generating..." during the request. An invite code string is displayed in bold. Expiration time is shown with a countdown (e.g., "23h 59m remaining"). A "Copy Code" button appears.
- **Requires Human**: Yes

### TC-02-14: Copy invite code to clipboard
- **Preconditions**: An invite code has been generated (TC-02-13 completed)
- **Steps**:
  1. Click "Copy Code"
- **Expected Result**: Button text changes to "Copied!" for 2 seconds, then reverts to "Copy Code". Code is in the clipboard.
- **Requires Human**: Yes

### TC-02-15: Join via invite code
- **Preconditions**: Second user is logged in with no babies, has a valid invite code from another user
- **Steps**:
  1. On the FirstLogin screen, click "Join with Invite Code"
  2. Enter the invite code
  3. Click "Join"
- **Expected Result**: Button shows "Joining..." during submission. On success, user is taken to the TodayDashboard for the joined baby.
- **Requires Human**: Yes

### TC-02-16: Join via invite code — empty code validation
- **Preconditions**: User is on the Join with Invite Code form
- **Steps**:
  1. Leave the invite code field empty
  2. Click "Join"
- **Expected Result**: Validation message "Invite code is required" appears. Form is not submitted.
- **Requires Human**: Yes

### TC-02-17: Join via invite code — invalid code
- **Preconditions**: User is on the Join with Invite Code form
- **Steps**:
  1. Enter an invalid or expired invite code
  2. Click "Join"
- **Expected Result**: Error message "Invalid or expired code" is displayed
- **Requires Human**: Yes

### TC-02-18: Unlink from baby — confirmation flow
- **Preconditions**: User is logged in, has a baby, on /settings
- **Steps**:
  1. Scroll to the "Unlink from Baby" section
  2. Click "Unlink from Baby" button
  3. Observe the confirmation dialog
- **Expected Result**: A confirmation dialog appears with message: "Are you sure you want to unlink from {babyName}? If you are the last linked parent, the baby and all associated data will be permanently deleted." Two buttons appear: "Confirm Unlink" and "Cancel".
- **Requires Human**: Yes

### TC-02-19: Unlink from baby — cancel
- **Preconditions**: Unlink confirmation dialog is showing (TC-02-18)
- **Steps**:
  1. Click "Cancel"
- **Expected Result**: Confirmation dialog closes. User remains on settings page. Baby is still linked.
- **Requires Human**: Yes

### TC-02-20: Unlink from baby — confirm
- **Preconditions**: Unlink confirmation dialog is showing, user has 2+ babies
- **Steps**:
  1. Click "Confirm Unlink"
- **Expected Result**: User is unlinked from the baby. The baby is removed from the baby list. Active baby switches to another linked baby.
- **Requires Human**: Yes

### TC-02-21: Account deletion — confirmation flow
- **Preconditions**: User is logged in, on /settings
- **Steps**:
  1. Scroll to the "Delete Account" section
  2. Click "Delete Account" button
- **Expected Result**: Confirmation dialog appears with message: "Are you sure you want to delete your account? This action cannot be undone. All your data will be permanently removed." Two buttons: "Confirm Delete" and "Cancel".
- **Requires Human**: Yes

### TC-02-22: Account deletion — cancel
- **Preconditions**: Account deletion confirmation dialog is showing
- **Steps**:
  1. Click "Cancel"
- **Expected Result**: Confirmation dialog closes. User remains on settings page.
- **Requires Human**: Yes

### TC-02-23: Account deletion — confirm
- **Preconditions**: Account deletion confirmation dialog is showing
- **Steps**:
  1. Click "Confirm Delete"
- **Expected Result**: Account is deleted. User is redirected to /login.
- **Requires Human**: Yes

---

## 3. Dashboard -- Today View

### TC-03-01: Dashboard loads and displays summary cards
- **Preconditions**: User is logged in with a baby that has today's data (feeds, stools, urine, temperature, weight)
- **Steps**:
  1. Navigate to /
- **Expected Result**: Six summary cards are displayed: Feeds (count), Calories (total), Wet Diapers (count), Stools (count with color indicator dot), Last Temp (value in degrees C or dash), Last Weight (value in kg or dash)
- **Requires Human**: Yes

### TC-03-02: Summary cards — no data state
- **Preconditions**: User is logged in with a baby that has no data logged today
- **Steps**:
  1. Navigate to /
- **Expected Result**: Summary cards show: Feeds=0, Calories=0, Wet Diapers=0, Stools=0, Last Temp="--", Last Weight="--"
- **Requires Human**: Yes

### TC-03-03: Stool color trend displays
- **Preconditions**: Baby has stool entries over the past 7 days
- **Steps**:
  1. Navigate to /
  2. Observe the "Stool Color Trend (7 days)" section
- **Expected Result**: Colored dots are displayed for each day with stool data. Each dot has a background color matching the stool status (red for ratings 1-3, amber for 4-5, green for 6-7). Hovering a dot shows a title with date, color label, and rating.
- **Requires Human**: Yes

### TC-03-04: Stool color trend — no data
- **Preconditions**: Baby has no stool entries in the past 7 days
- **Steps**:
  1. Navigate to /
- **Expected Result**: The "Stool Color Trend (7 days)" section is not displayed
- **Requires Human**: Yes

### TC-03-05: Upcoming medications display
- **Preconditions**: Baby has active medications with schedule_times set
- **Steps**:
  1. Navigate to /
  2. Observe the "Upcoming Medications" section
- **Expected Result**: Each medication shows its name, dose, and a countdown timer (e.g., "in 2 h 30 min"). If a dose is overdue, it shows "overdue by X h Y min".
- **Requires Human**: Yes

### TC-03-06: Upcoming medications — overdue dose
- **Preconditions**: Baby has a medication with a scheduled time that has passed
- **Steps**:
  1. Navigate to /
- **Expected Result**: The overdue medication shows countdown in red-style text like "overdue by 45 min"
- **Requires Human**: Yes

### TC-03-07: Upcoming medications — no schedule
- **Preconditions**: Baby has a medication with frequency "as_needed" (no scheduled times)
- **Steps**:
  1. Navigate to /
- **Expected Result**: That medication's countdown area shows "No schedule"
- **Requires Human**: Yes

### TC-03-08: Upcoming medications — none configured
- **Preconditions**: Baby has no active medications
- **Steps**:
  1. Navigate to /
- **Expected Result**: The "Upcoming Medications" section is not displayed
- **Requires Human**: Yes

### TC-03-09: Alert banner — acholic stool
- **Preconditions**: Baby has a recent stool entry with color_rating 1-3 (triggers acholic_stool alert)
- **Steps**:
  1. Navigate to /
- **Expected Result**: An alert banner appears with label "Acholic Stool" and message containing "Contact your hepatology team -- this may indicate bile flow failure."
- **Requires Human**: Yes

### TC-03-10: Alert banner — fever
- **Preconditions**: Baby has a recent temperature entry exceeding the fever threshold
- **Steps**:
  1. Navigate to /
- **Expected Result**: An alert banner appears with label "Fever" and message containing "Contact your hepatology team immediately. Fever after Kasai can indicate cholangitis."
- **Requires Human**: Yes

### TC-03-11: Alert banner — jaundice worsening
- **Preconditions**: Backend has detected worsening jaundice and returns a jaundice_worsening alert
- **Steps**:
  1. Navigate to /
- **Expected Result**: An alert banner appears with label "Jaundice Worsening" and message "Worsening jaundice detected. Contact your hepatology team."
- **Requires Human**: Yes

### TC-03-12: Alert banner — missed medication
- **Preconditions**: A scheduled medication dose has been missed
- **Steps**:
  1. Navigate to /
- **Expected Result**: An alert banner appears with label "Missed Medication" and message "A scheduled medication dose was missed. Tap to log."
- **Requires Human**: Yes

### TC-03-13: Alert dismissal
- **Preconditions**: One or more alert banners are showing on the dashboard
- **Steps**:
  1. Click "Dismiss" on an alert banner
- **Expected Result**: The alert banner disappears from the page
- **Requires Human**: Yes

### TC-03-14: Alert dismissal persists across page loads (localStorage)
- **Preconditions**: An alert has been dismissed (TC-03-13 completed)
- **Steps**:
  1. Refresh the page (or navigate away and back to /)
- **Expected Result**: The previously dismissed alert remains hidden. Other non-dismissed alerts still appear.
- **Requires Human**: Yes

### TC-03-15: Dismissed alerts cleaned up when alert no longer active
- **Preconditions**: An alert was dismissed in a previous session, and the underlying condition has since been resolved
- **Steps**:
  1. Navigate to /
- **Expected Result**: The stale dismissed alert ID is removed from localStorage (cleanup logic runs). No orphaned dismiss entries remain.
- **Requires Human**: Yes

### TC-03-16: Dashboard loading state
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /
  2. Observe the initial render
- **Expected Result**: A "Loading..." message appears while the dashboard API call is in progress, then the dashboard content replaces it
- **Requires Human**: Yes

### TC-03-17: Dashboard error state
- **Preconditions**: User is logged in with a baby, API returns an error
- **Steps**:
  1. Navigate to / with the API endpoint unreachable or returning 500
- **Expected Result**: Error message "Failed to load dashboard data" is displayed instead of dashboard content
- **Requires Human**: Yes

### TC-03-18: Medication countdown updates every minute
- **Preconditions**: Dashboard is displayed with upcoming medications
- **Steps**:
  1. Observe the medication countdown value
  2. Wait 1+ minutes
- **Expected Result**: The countdown value updates automatically (the timer interval is 60 seconds)
- **Requires Human**: Yes

---

## 4. Metric Entry Forms

### 4.1 Feeding

### TC-04-01: Feeding form renders
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/feeding
- **Expected Result**: Form displays: Timestamp (datetime-local, pre-filled with current time), Feed type dropdown (Breast Milk, Formula, Fortified Breast Milk, Solid, Other), Volume (mL) number input, Caloric density (kcal/oz) number input, Duration (min) number input, Notes textarea, "Log Feeding" submit button. A "Back" link to / is at the top. Page heading says "Log Feeding".
- **Requires Human**: Yes

### TC-04-02: Feeding form — required field validation
- **Preconditions**: On /log/feeding
- **Steps**:
  1. Leave Feed type as "Select..."
  2. Click "Log Feeding"
- **Expected Result**: Validation message "Feed type is required" appears. Form is not submitted.
- **Requires Human**: Yes

### TC-04-03: Feeding form — successful submission
- **Preconditions**: On /log/feeding
- **Steps**:
  1. Select "Formula" as feed type
  2. Enter 120 for Volume (mL)
  3. Enter 20 for Caloric density
  4. Click "Log Feeding"
- **Expected Result**: Button shows "Logging..." during submission. On success, user is redirected to /. Entry is visible on the dashboard (feed count increments).
- **Requires Human**: Yes

### TC-04-04: Feeding form — edit timestamp
- **Preconditions**: On /log/feeding
- **Steps**:
  1. Change the Timestamp field to a past time
  2. Fill in required fields
  3. Click "Log Feeding"
- **Expected Result**: Entry is saved with the manually set timestamp, not the current time
- **Requires Human**: Yes

### TC-04-05: Feeding form — optional fields omitted
- **Preconditions**: On /log/feeding
- **Steps**:
  1. Select a feed type
  2. Leave Volume, Caloric density, Duration, and Notes empty
  3. Click "Log Feeding"
- **Expected Result**: Entry is saved successfully with only timestamp and feed_type
- **Requires Human**: Yes

### 4.2 Urine

### TC-04-06: Urine form renders
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/urine
- **Expected Result**: Form displays: Timestamp (pre-filled), Color dropdown (Clear, Pale Yellow, Dark Yellow, Amber, Brown), Notes textarea, "Log Urine" button
- **Requires Human**: Yes

### TC-04-07: Urine form — successful submission (no required fields besides timestamp)
- **Preconditions**: On /log/urine
- **Steps**:
  1. Click "Log Urine" without changing anything
- **Expected Result**: Entry is saved with timestamp only. User is redirected to /. Wet diaper count increments.
- **Requires Human**: Yes

### TC-04-08: Urine form — with optional fields
- **Preconditions**: On /log/urine
- **Steps**:
  1. Select "Dark Yellow" for Color
  2. Enter notes text
  3. Click "Log Urine"
- **Expected Result**: Entry is saved with color and notes fields populated
- **Requires Human**: Yes

### 4.3 Stool

### TC-04-09: Stool form renders with color swatches
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/stool
- **Expected Result**: Form displays: Timestamp, a "Stool Color" fieldset with 7 colored swatch buttons (White, Clay, Pale Yellow, Yellow, Light Green, Green, Brown), Consistency dropdown (Watery, Loose, Soft, Formed, Hard), Volume estimate dropdown (Small, Medium, Large), Photo upload component showing "0 / 4 photos", Notes textarea, "Log Stool" button
- **Requires Human**: Yes

### TC-04-10: Stool form — color swatch selection
- **Preconditions**: On /log/stool
- **Steps**:
  1. Click the "Green" (rating 6) color swatch button
- **Expected Result**: The selected swatch gets a 3px solid black border. aria-pressed becomes "true" for the selected swatch and "false" for all others.
- **Requires Human**: Yes

### TC-04-11: Stool form — acholic stool warning (ratings 1-3)
- **Preconditions**: On /log/stool
- **Steps**:
  1. Click the "White" (rating 1) color swatch
- **Expected Result**: A red bold warning appears: "Warning: Acholic stool detected (color 1). Contact your hepatology team."
- **Requires Human**: Yes

### TC-04-12: Stool form — acholic warning for Clay (rating 2)
- **Preconditions**: On /log/stool
- **Steps**:
  1. Click the "Clay" (rating 2) color swatch
- **Expected Result**: Warning message appears: "Warning: Acholic stool detected (color 2). Contact your hepatology team."
- **Requires Human**: Yes

### TC-04-13: Stool form — acholic warning for Pale Yellow (rating 3)
- **Preconditions**: On /log/stool
- **Steps**:
  1. Click the "Pale Yellow" (rating 3) color swatch
- **Expected Result**: Warning message appears: "Warning: Acholic stool detected (color 3). Contact your hepatology team."
- **Requires Human**: Yes

### TC-04-14: Stool form — no warning for rating 4+
- **Preconditions**: On /log/stool
- **Steps**:
  1. Click the "Yellow" (rating 4) color swatch
- **Expected Result**: No acholic stool warning appears
- **Requires Human**: Yes

### TC-04-15: Stool form — validation: no color selected
- **Preconditions**: On /log/stool
- **Steps**:
  1. Do not select any color swatch
  2. Click "Log Stool"
- **Expected Result**: Validation message "Stool color is required" appears. Form is not submitted.
- **Requires Human**: Yes

### TC-04-16: Stool form — photo upload
- **Preconditions**: On /log/stool
- **Steps**:
  1. Click the Photo file input
  2. Select a JPEG image
- **Expected Result**: "Uploading..." text appears during upload. After success, the photo count updates to "1 / 4 photos".
- **Requires Human**: Yes (file selection)

### TC-04-17: Stool form — multiple photo upload (max 4)
- **Preconditions**: On /log/stool, 3 photos already uploaded
- **Steps**:
  1. Select 2 more photos at once
- **Expected Result**: Only 1 of the 2 photos is uploaded (reaching the limit of 4). A warning appears: "Only 1 of 2 photos uploaded (limit: 4)". Counter shows "4 / 4 photos". File input becomes disabled.
- **Requires Human**: Yes (file selection)

### TC-04-18: Stool form — successful submission
- **Preconditions**: On /log/stool
- **Steps**:
  1. Select a color swatch (e.g., Green, rating 6)
  2. Select "Soft" for Consistency
  3. Select "Medium" for Volume
  4. Add a note
  5. Click "Log Stool"
- **Expected Result**: Entry is saved. User is redirected to /. Stool count increments on dashboard.
- **Requires Human**: Yes

### 4.4 Temperature

### TC-04-19: Temperature form renders
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/temperature
- **Expected Result**: Form displays: Timestamp, Temperature (degrees C) number input (step 0.1, min 30, max 45), Method dropdown (Rectal, Axillary, Ear, Forehead), Notes textarea, "Log Temperature" button
- **Requires Human**: Yes

### TC-04-20: Temperature form — validation: missing temperature
- **Preconditions**: On /log/temperature
- **Steps**:
  1. Select a method but leave temperature empty
  2. Click "Log Temperature"
- **Expected Result**: Validation message "Temperature is required" appears
- **Requires Human**: Yes

### TC-04-21: Temperature form — validation: missing method
- **Preconditions**: On /log/temperature
- **Steps**:
  1. Enter a temperature value but leave method as "Select..."
  2. Click "Log Temperature"
- **Expected Result**: Validation message "Method is required" appears
- **Requires Human**: Yes

### TC-04-22: Temperature form — fever warning (rectal >= 38.0)
- **Preconditions**: On /log/temperature
- **Steps**:
  1. Enter 38.5 for Temperature
  2. Select "Rectal" for Method
- **Expected Result**: A fever warning appears immediately (before submission): "Fever detected. Contact your hepatology team immediately. Fever after Kasai can indicate cholangitis."
- **Requires Human**: Yes

### TC-04-23: Temperature form — fever warning (axillary >= 37.5)
- **Preconditions**: On /log/temperature
- **Steps**:
  1. Enter 37.5 for Temperature
  2. Select "Axillary" for Method
- **Expected Result**: Fever warning appears with the same cholangitis message
- **Requires Human**: Yes

### TC-04-24: Temperature form — no fever warning below threshold
- **Preconditions**: On /log/temperature
- **Steps**:
  1. Enter 37.0 for Temperature
  2. Select "Rectal" for Method
- **Expected Result**: No fever warning is displayed (37.0 < 38.0 rectal threshold)
- **Requires Human**: Yes

### TC-04-25: Temperature form — successful submission
- **Preconditions**: On /log/temperature
- **Steps**:
  1. Enter 37.2 for Temperature
  2. Select "Axillary" for Method
  3. Click "Log Temperature"
- **Expected Result**: Entry is saved. User is redirected to /. Dashboard "Last Temp" card shows 37.2 degrees C.
- **Requires Human**: Yes

### 4.5 Weight

### TC-04-26: Weight form renders
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/weight
- **Expected Result**: Form displays: Timestamp, Weight (kg) number input (step 0.01), Measurement source dropdown (Home Scale, Clinic), Notes textarea, "Log Weight" button
- **Requires Human**: Yes

### TC-04-27: Weight form — validation: missing weight
- **Preconditions**: On /log/weight
- **Steps**:
  1. Leave Weight empty
  2. Click "Log Weight"
- **Expected Result**: Validation message "Weight is required" appears
- **Requires Human**: Yes

### TC-04-28: Weight form — successful submission
- **Preconditions**: On /log/weight
- **Steps**:
  1. Enter 5.45 for Weight
  2. Select "Clinic" for Measurement source
  3. Click "Log Weight"
- **Expected Result**: Entry is saved. User is redirected to /. Dashboard "Last Weight" card shows 5.45 kg.
- **Requires Human**: Yes

### 4.6 Abdomen

### TC-04-29: Abdomen form renders
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/abdomen
- **Expected Result**: Form displays: Timestamp, Firmness dropdown (Soft, Firm, Distended), Tenderness checkbox, Girth (cm) number input (step 0.1), Photo upload (multi, 0/4 photos), Notes textarea, "Log Abdomen" button
- **Requires Human**: Yes

### TC-04-30: Abdomen form — validation: missing firmness
- **Preconditions**: On /log/abdomen
- **Steps**:
  1. Leave Firmness as "Select..."
  2. Click "Log Abdomen"
- **Expected Result**: Validation message "Firmness is required" appears
- **Requires Human**: Yes

### TC-04-31: Abdomen form — successful submission with all fields
- **Preconditions**: On /log/abdomen
- **Steps**:
  1. Select "Distended" for Firmness
  2. Check Tenderness
  3. Enter 38.5 for Girth
  4. Add a note
  5. Click "Log Abdomen"
- **Expected Result**: Entry is saved with all fields. User is redirected to /.
- **Requires Human**: Yes

### TC-04-32: Abdomen form — photo upload
- **Preconditions**: On /log/abdomen
- **Steps**:
  1. Upload a photo via the file input
- **Expected Result**: Photo uploads successfully. Counter updates to "1 / 4 photos".
- **Requires Human**: Yes (file selection)

### 4.7 Skin

### TC-04-33: Skin form renders
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/skin
- **Expected Result**: Form displays: Timestamp, Jaundice level dropdown (None, Mild (Face), Moderate (Trunk), Severe (Limbs & Trunk)), Scleral icterus checkbox, Rashes text input, Bruising text input, Photo upload (multi, with hint "Consistent lighting recommended"), Notes textarea, "Log Skin" button
- **Requires Human**: Yes

### TC-04-34: Skin form — no required field validation (all optional except timestamp)
- **Preconditions**: On /log/skin
- **Steps**:
  1. Click "Log Skin" without filling any fields
- **Expected Result**: Entry is saved with only timestamp and scleral_icterus=false. User is redirected to /.
- **Requires Human**: Yes

### TC-04-35: Skin form — successful submission with all fields
- **Preconditions**: On /log/skin
- **Steps**:
  1. Select "Moderate (Trunk)" for Jaundice level
  2. Check Scleral icterus
  3. Enter "Eczema on arms" for Rashes
  4. Enter "Small bruise on leg" for Bruising
  5. Upload a photo
  6. Enter notes
  7. Click "Log Skin"
- **Expected Result**: Entry is saved with all fields populated. User is redirected to /.
- **Requires Human**: Yes

### TC-04-36: Skin form — photo hint text displayed
- **Preconditions**: On /log/skin
- **Steps**:
  1. Observe the photo upload area
- **Expected Result**: The hint "Consistent lighting recommended" is displayed near the photo upload input
- **Requires Human**: Yes

### 4.8 Bruising

### TC-04-37: Bruising form renders
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/bruising
- **Expected Result**: Form displays: Timestamp, Location on body text input (placeholder "e.g., left arm, torso"), Size estimate dropdown (Small <1cm, Medium 1-3cm, Large >3cm), Size (cm) number input (step 0.1), Color text input (placeholder "e.g., red, purple, yellow-green"), Photo upload (multi, 0/4), Notes textarea, "Log Bruising" button
- **Requires Human**: Yes

### TC-04-38: Bruising form — validation: missing location
- **Preconditions**: On /log/bruising
- **Steps**:
  1. Leave Location empty
  2. Select a size estimate
  3. Click "Log Bruising"
- **Expected Result**: Validation message "Location is required" appears
- **Requires Human**: Yes

### TC-04-39: Bruising form — validation: missing size estimate
- **Preconditions**: On /log/bruising
- **Steps**:
  1. Enter a location
  2. Leave Size estimate as "Select..."
  3. Click "Log Bruising"
- **Expected Result**: Validation message "Size estimate is required" appears
- **Requires Human**: Yes

### TC-04-40: Bruising form — successful submission
- **Preconditions**: On /log/bruising
- **Steps**:
  1. Enter "Right forearm" for Location
  2. Select "Medium (1-3cm)" for Size estimate
  3. Enter 2.5 for Size (cm)
  4. Enter "Purple" for Color
  5. Click "Log Bruising"
- **Expected Result**: Entry is saved. User is redirected to /.
- **Requires Human**: Yes

### 4.9 Lab

### TC-04-41: Lab form renders with quick picks
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/lab
- **Expected Result**: Form displays: Timestamp, a "Quick Pick" fieldset with 8 buttons (Total Bilirubin, Direct Bilirubin, ALT, AST, GGT, Albumin, INR, Platelets), Test name text input, Value text input, Unit text input, Normal range text input (placeholder "e.g., 0.1-1.2"), Notes textarea, "Log Lab" button
- **Requires Human**: Yes

### TC-04-42: Lab form — quick pick selection
- **Preconditions**: On /log/lab
- **Steps**:
  1. Click the "Total Bilirubin" quick pick button
- **Expected Result**: Test name field is populated with "total_bilirubin". Unit field is populated with "mg/dL". The button shows aria-pressed="true".
- **Requires Human**: Yes

### TC-04-43: Lab form — quick pick changes unit automatically
- **Preconditions**: On /log/lab
- **Steps**:
  1. Click "ALT" quick pick
  2. Observe Unit field
  3. Click "INR" quick pick
  4. Observe Unit field
- **Expected Result**: After ALT: unit is "U/L". After INR: unit is empty string (INR is unitless). Test name updates each time.
- **Requires Human**: Yes

### TC-04-44: Lab form — validation: missing test name
- **Preconditions**: On /log/lab
- **Steps**:
  1. Enter a value but leave Test name empty
  2. Click "Log Lab"
- **Expected Result**: Validation message "Test name is required" appears
- **Requires Human**: Yes

### TC-04-45: Lab form — validation: missing value
- **Preconditions**: On /log/lab
- **Steps**:
  1. Enter a test name but leave Value empty
  2. Click "Log Lab"
- **Expected Result**: Validation message "Value is required" appears
- **Requires Human**: Yes

### TC-04-46: Lab form — successful submission
- **Preconditions**: On /log/lab
- **Steps**:
  1. Click "GGT" quick pick
  2. Enter "85" for Value
  3. Enter "9-48" for Normal range
  4. Click "Log Lab"
- **Expected Result**: Entry is saved with test_name="GGT", value="85", unit="U/L", normal_range="9-48". User is redirected to /.
- **Requires Human**: Yes

### TC-04-47: Lab form — manual test name entry (no quick pick)
- **Preconditions**: On /log/lab
- **Steps**:
  1. Type "Vitamin D" in the Test name field
  2. Enter "32" for Value
  3. Type "ng/mL" for Unit
  4. Click "Log Lab"
- **Expected Result**: Entry is saved with the manually entered test name, value, and unit
- **Requires Human**: Yes

### 4.10 General Notes

### TC-04-48: Notes form renders
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/notes
- **Expected Result**: Form displays: Timestamp, Content textarea, Category dropdown (Behavior, Sleep, Vomiting, Irritability, Skin, Other), Photo upload (multi), photo counter "0 / 4 photos", "Log Note" button
- **Requires Human**: Yes

### TC-04-49: Notes form — validation: missing content
- **Preconditions**: On /log/notes
- **Steps**:
  1. Leave Content empty
  2. Click "Log Note"
- **Expected Result**: Validation message "Content is required" appears
- **Requires Human**: Yes

### TC-04-50: Notes form — successful submission
- **Preconditions**: On /log/notes
- **Steps**:
  1. Enter "Baby seemed more irritable today after afternoon nap" in Content
  2. Select "Irritability" for Category
  3. Click "Log Note"
- **Expected Result**: Entry is saved. User is redirected to /.
- **Requires Human**: Yes

### TC-04-51: Notes form — photo upload
- **Preconditions**: On /log/notes
- **Steps**:
  1. Upload a photo
- **Expected Result**: Photo uploads. Counter updates. Photo is attached to the entry on submission.
- **Requires Human**: Yes (file selection)

### 4.11 Common Form Behaviors

### TC-04-52: All forms — back link navigates to home
- **Preconditions**: On any /log/[metric] page
- **Steps**:
  1. Click the "Back" link at the top
- **Expected Result**: User is navigated to /
- **Requires Human**: Yes

### TC-04-53: All forms — no baby selected state
- **Preconditions**: User is logged in but has no active baby
- **Steps**:
  1. Navigate to /log/feeding
- **Expected Result**: Message "No baby selected" is displayed instead of the form
- **Requires Human**: Yes

### TC-04-54: Unknown metric type
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /log/invalidmetric
- **Expected Result**: Message "Unknown metric type" is displayed
- **Requires Human**: Yes

### TC-04-55: Form state resets when switching metrics
- **Preconditions**: On /log/feeding with some fields filled
- **Steps**:
  1. Navigate to /log/stool
- **Expected Result**: The stool form loads fresh with no leftover state. photoKeys, error, submitting, and uploading are all reset.
- **Requires Human**: Yes

### TC-04-56: Photo upload — accepted file types
- **Preconditions**: On any form with photo upload (stool, abdomen, skin, bruising, notes)
- **Steps**:
  1. Observe the file input accept attribute
- **Expected Result**: The file input accepts "image/jpeg,image/png,image/heic" formats
- **Requires Human**: No

### TC-04-57: Photo upload — upload failure
- **Preconditions**: On /log/stool, API photo upload returns an error
- **Steps**:
  1. Select a photo to upload
- **Expected Result**: Error message "Photo upload failed" is displayed. Photo count does not increment.
- **Requires Human**: Yes

### TC-04-58: Server error on form submission
- **Preconditions**: On any /log/[metric] form, API returns an error on POST
- **Steps**:
  1. Fill in required fields
  2. Submit the form
- **Expected Result**: Error message "Failed to save" (or the server error message) is displayed. User remains on the form page. Form data is preserved.
- **Requires Human**: Yes

---

## 5. Medication Management

### TC-05-01: Medication list page renders
- **Preconditions**: User is logged in with a baby, on /medications
- **Steps**:
  1. Navigate to /medications
- **Expected Result**: Page shows "Medications" heading, a "Back" link, the medication list (or empty state), and an "Add Medication" button
- **Requires Human**: Yes

### TC-05-02: Medication list — empty state
- **Preconditions**: Baby has no medications
- **Steps**:
  1. Navigate to /medications
- **Expected Result**: Message "No medications found." is displayed. "Add Medication" button is still present.
- **Requires Human**: Yes

### TC-05-03: Medication list — displays medications with details
- **Preconditions**: Baby has medications configured
- **Steps**:
  1. Navigate to /medications
- **Expected Result**: Each medication shows: name, dose, formatted frequency. Action buttons for each: "Deactivate" (or "Reactivate" if inactive), "Edit", "View Logs". Inactive medications have a visual "inactive" class distinction.
- **Requires Human**: Yes

### TC-05-04: Create medication — navigate from list
- **Preconditions**: On /medications
- **Steps**:
  1. Click "Add Medication"
- **Expected Result**: User is navigated to /log/medication. Form heading shows "Log Medication".
- **Requires Human**: Yes

### TC-05-05: Create medication — form fields
- **Preconditions**: On /log/medication (not editing)
- **Steps**:
  1. Observe the form
- **Expected Result**: Form displays: Medication Name text input with datalist suggestions (UDCA, Bactrim, Vitamins A/D/E/K, Iron, Other), Dose text input, Frequency dropdown (Once daily, Twice daily, Three times daily, As needed, Custom), Notes textarea, "Save Medication" button
- **Requires Human**: Yes

### TC-05-06: Create medication — medication name suggestions
- **Preconditions**: On /log/medication
- **Steps**:
  1. Click into the Medication Name field
  2. Start typing "UDCA"
- **Expected Result**: Datalist suggestions appear including "UDCA (ursodiol)"
- **Requires Human**: Yes

### TC-05-07: Create medication — frequency-based schedule time slots
- **Preconditions**: On /log/medication
- **Steps**:
  1. Select "Twice daily" for Frequency
- **Expected Result**: Two "Schedule Time" time inputs appear (Schedule Time 1, Schedule Time 2)
- **Requires Human**: Yes

### TC-05-08: Create medication — frequency "As needed" hides schedule
- **Preconditions**: On /log/medication
- **Steps**:
  1. Select "As needed" for Frequency
- **Expected Result**: No schedule time inputs are displayed
- **Requires Human**: Yes

### TC-05-09: Create medication — custom frequency with add time button
- **Preconditions**: On /log/medication
- **Steps**:
  1. Select "Custom" for Frequency
  2. Observe initial state
  3. Click "Add Time" button
- **Expected Result**: Initially, one schedule time slot appears. After clicking "Add Time", a second time slot is added. Button allows adding more.
- **Requires Human**: Yes

### TC-05-10: Create medication — validation: missing name
- **Preconditions**: On /log/medication
- **Steps**:
  1. Leave Name empty, fill in Dose and Frequency
  2. Click "Save Medication"
- **Expected Result**: Validation message "Medication name is required" appears
- **Requires Human**: Yes

### TC-05-11: Create medication — validation: missing dose
- **Preconditions**: On /log/medication
- **Steps**:
  1. Fill in Name, leave Dose empty, select Frequency
  2. Click "Save Medication"
- **Expected Result**: Validation message "Dose is required" appears
- **Requires Human**: Yes

### TC-05-12: Create medication — validation: missing frequency
- **Preconditions**: On /log/medication
- **Steps**:
  1. Fill in Name and Dose, leave Frequency as "Select..."
  2. Click "Save Medication"
- **Expected Result**: Validation message "Frequency is required" appears
- **Requires Human**: Yes

### TC-05-13: Create medication — successful submission
- **Preconditions**: On /log/medication
- **Steps**:
  1. Enter "UDCA (ursodiol)" for Name
  2. Enter "45mg" for Dose
  3. Select "Twice daily" for Frequency
  4. Set schedule times to 08:00 and 20:00
  5. Click "Save Medication"
- **Expected Result**: Button shows "Saving..." during submission. Medication is created. User is redirected to /.
- **Requires Human**: Yes

### TC-05-14: Edit medication — navigate from list
- **Preconditions**: On /medications, at least one medication exists
- **Steps**:
  1. Click "Edit" on a medication
- **Expected Result**: User is navigated to /log/medication?edit={medicationId}. Page heading shows "Edit Medication". Form fields are pre-populated with the medication's current data.
- **Requires Human**: Yes

### TC-05-15: Edit medication — loading state
- **Preconditions**: Navigating to /log/medication?edit={id}
- **Steps**:
  1. Observe the page while medication data loads
- **Expected Result**: "Loading medication..." message is displayed until data is fetched, then the form appears with pre-filled values
- **Requires Human**: Yes

### TC-05-16: Edit medication — save changes
- **Preconditions**: On /log/medication?edit={id}, form is pre-populated
- **Steps**:
  1. Change the Dose to a new value
  2. Click "Save Medication"
- **Expected Result**: Medication is updated via PUT API call. User is redirected to /medications (not / like new entries).
- **Requires Human**: Yes

### TC-05-17: Deactivate medication
- **Preconditions**: On /medications, an active medication exists
- **Steps**:
  1. Click "Deactivate" on an active medication
- **Expected Result**: Medication's active status is toggled to inactive. The list refreshes. The medication now shows a "Reactivate" button and has the "inactive" visual style.
- **Requires Human**: Yes

### TC-05-18: Reactivate medication
- **Preconditions**: On /medications, an inactive medication exists
- **Steps**:
  1. Click "Reactivate" on an inactive medication
- **Expected Result**: Medication's active status is toggled to active. The list refreshes. The medication now shows a "Deactivate" button and normal visual style.
- **Requires Human**: Yes

### TC-05-19: Log dose as given
- **Preconditions**: On /log/med, at least one active medication exists
- **Steps**:
  1. Select a medication from the dropdown
  2. Click the "Given" status button
  3. Click "Log Dose"
- **Expected Result**: Button shows "Logging..." during submission. Dose is logged as given (skipped=false). User is redirected to /.
- **Requires Human**: Yes

### TC-05-20: Log dose as skipped — with reason
- **Preconditions**: On /log/med, at least one active medication exists
- **Steps**:
  1. Select a medication
  2. Click the "Skipped" status button
  3. Observe that a "Skip Reason" input appears
  4. Enter "Baby vomited" as the skip reason
  5. Click "Log Dose"
- **Expected Result**: Dose is logged as skipped with the skip reason. User is redirected to /.
- **Requires Human**: Yes

### TC-05-21: Log dose — skip reason hidden when "Given" selected
- **Preconditions**: On /log/med
- **Steps**:
  1. Click "Skipped" (skip reason input appears)
  2. Click "Given"
- **Expected Result**: The "Skip Reason" input disappears
- **Requires Human**: Yes

### TC-05-22: Dose log — validation: no medication selected
- **Preconditions**: On /log/med
- **Steps**:
  1. Leave medication dropdown as "Select medication..."
  2. Click "Given"
  3. Click "Log Dose"
- **Expected Result**: Validation message "Medication is required" appears
- **Requires Human**: Yes

### TC-05-23: Dose log — validation: no status selected
- **Preconditions**: On /log/med
- **Steps**:
  1. Select a medication
  2. Do not click "Given" or "Skipped"
  3. Click "Log Dose"
- **Expected Result**: Validation message "Select Given or Skipped" appears
- **Requires Human**: Yes

### TC-05-24: Dose log — deep link with medication_id
- **Preconditions**: User has an active medication
- **Steps**:
  1. Navigate to /log/med?medication_id={known_medication_id}
- **Expected Result**: The DoseLogForm loads with the medication pre-selected in the dropdown
- **Requires Human**: Yes

### TC-05-25: Dose log — deep link with scheduled_time
- **Preconditions**: User has an active medication
- **Steps**:
  1. Navigate to /log/med?medication_id={id}&scheduled_time={iso_timestamp}
  2. Submit the dose log
- **Expected Result**: The scheduled_time is included in the payload sent to the API
- **Requires Human**: Yes

### TC-05-26: Dose log — only active medications shown in dropdown
- **Preconditions**: Baby has both active and inactive medications
- **Steps**:
  1. Navigate to /log/med
  2. Open the medication dropdown
- **Expected Result**: Only active medications appear in the dropdown. Inactive medications are filtered out.
- **Requires Human**: Yes

### TC-05-27: Dose log — medication load failure
- **Preconditions**: On /log/med, API medications endpoint returns error
- **Steps**:
  1. Navigate to /log/med
- **Expected Result**: Error message "Failed to load medications" appears
- **Requires Human**: Yes

### TC-05-28: View medication logs from list
- **Preconditions**: On /medications, medications exist
- **Steps**:
  1. Click "View Logs" on a medication
- **Expected Result**: User is navigated to /log/med?medication_id={medicationId}
- **Requires Human**: Yes

---

## 6. Trends View

### TC-06-01: Trends page loads with default 7-day range
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /trends
- **Expected Result**: DateRangeSelector is shown with "7d" selected. Seven chart sections appear: Stool Color, Weight, Temperature, Abdomen Girth, Feeding, Diaper Counts, Lab Trends. Loading indicator shows while data is fetched.
- **Requires Human**: Yes

### TC-06-02: Date range selector — 14d
- **Preconditions**: On /trends
- **Steps**:
  1. Select "14d" in the DateRangeSelector
- **Expected Result**: Charts reload with data from the past 14 days. Loading indicator appears during fetch.
- **Requires Human**: Yes

### TC-06-03: Date range selector — 30d
- **Preconditions**: On /trends
- **Steps**:
  1. Select "30d"
- **Expected Result**: Charts reload with 30-day data range
- **Requires Human**: Yes

### TC-06-04: Date range selector — 90d
- **Preconditions**: On /trends
- **Steps**:
  1. Select "90d"
- **Expected Result**: Charts reload with 90-day data range
- **Requires Human**: Yes

### TC-06-05: Date range selector — custom range
- **Preconditions**: On /trends
- **Steps**:
  1. Select "custom" in the DateRangeSelector
  2. Enter a From date and To date
- **Expected Result**: Charts reload with data from the custom date range
- **Requires Human**: Yes

### TC-06-06: Weight chart — WHO percentile overlay
- **Preconditions**: On /trends, baby has weight data
- **Steps**:
  1. Observe the Weight chart section
- **Expected Result**: The weight chart displays data points along with WHO percentile curves overlaid. Percentiles are fetched based on the baby's sex and age range within the selected date window.
- **Requires Human**: Yes

### TC-06-07: Temperature chart — fever threshold line
- **Preconditions**: On /trends, baby has temperature data
- **Steps**:
  1. Observe the Temperature chart section
- **Expected Result**: The temperature chart shows data points with a fever threshold reference line
- **Requires Human**: Yes

### TC-06-08: Stool color chart
- **Preconditions**: On /trends, baby has stool data
- **Steps**:
  1. Observe the Stool Color chart section
- **Expected Result**: Chart displays stool color scores over time
- **Requires Human**: Yes

### TC-06-09: Abdomen girth chart
- **Preconditions**: On /trends, baby has abdomen measurements
- **Steps**:
  1. Observe the Abdomen Girth chart section
- **Expected Result**: Chart shows girth_cm measurements over time
- **Requires Human**: Yes

### TC-06-10: Feeding chart
- **Preconditions**: On /trends, baby has feeding data
- **Steps**:
  1. Observe the Feeding chart section
- **Expected Result**: Chart shows daily feeding data (total volume, calories, feed count, breakdown by type)
- **Requires Human**: Yes

### TC-06-11: Diaper chart
- **Preconditions**: On /trends, baby has diaper data
- **Steps**:
  1. Observe the Diaper Counts chart section
- **Expected Result**: Chart shows daily wet_count and stool_count
- **Requires Human**: Yes

### TC-06-12: Lab trends chart
- **Preconditions**: On /trends, baby has lab entries
- **Steps**:
  1. Observe the Lab Trends chart section
- **Expected Result**: Chart shows lab values organized by test name over time
- **Requires Human**: Yes

### TC-06-13: Trends — loading state
- **Preconditions**: On /trends
- **Steps**:
  1. Observe the page during initial data load
- **Expected Result**: "Loading..." text is displayed while the API call is in progress
- **Requires Human**: Yes

### TC-06-14: Trends — error state
- **Preconditions**: On /trends, API returns an error
- **Steps**:
  1. Navigate to /trends with backend unavailable
- **Expected Result**: Error message "Failed to load trends data" is displayed
- **Requires Human**: Yes

### TC-06-15: Trends — no baby selected
- **Preconditions**: User is logged in but has no active baby
- **Steps**:
  1. Navigate to /trends
- **Expected Result**: Message "No baby selected" is displayed
- **Requires Human**: Yes

### TC-06-16: Trends — WHO percentile fetch failure is graceful
- **Preconditions**: On /trends, WHO percentile API endpoint fails
- **Steps**:
  1. Navigate to /trends
- **Expected Result**: Weight chart still renders with data points, but without the percentile overlay curves. No error message is shown to the user (the percentile fetch failure is caught silently).
- **Requires Human**: Yes

---

## 7. Reports

### TC-07-01: Report page renders
- **Preconditions**: User is logged in with a baby
- **Steps**:
  1. Navigate to /report
- **Expected Result**: Page shows "Clinical Report" heading. Two date inputs: From and To. A "Generate Report" button (disabled until dates are valid). Content preview section is hidden until dates are set.
- **Requires Human**: Yes

### TC-07-02: Report — date selection enables generate button
- **Preconditions**: On /report
- **Steps**:
  1. Select a From date
  2. Select a To date (after From)
- **Expected Result**: The "Generate Report" button becomes enabled. A preview summary appears showing: baby name, formatted date range, and a list of included report sections (stool color log, weight chart, lab trends, temperature log, feeding summary, medication adherence, notable observations and photos).
- **Requires Human**: Yes

### TC-07-03: Report — invalid date range (To before From)
- **Preconditions**: On /report
- **Steps**:
  1. Select a From date
  2. Select a To date that is before the From date
- **Expected Result**: "Generate Report" button remains disabled. No preview summary is shown.
- **Requires Human**: Yes

### TC-07-04: Report — PDF generation and download
- **Preconditions**: On /report, valid dates selected
- **Steps**:
  1. Click "Generate Report"
- **Expected Result**: Button text changes to "Generating..." and is disabled during the request. A PDF file is downloaded with filename format "report-{babyname}-{from}-to-{to}.pdf". Button returns to "Generate Report" after completion.
- **Requires Human**: Yes

### TC-07-05: Report — generation error
- **Preconditions**: On /report, valid dates selected, API returns an error
- **Steps**:
  1. Click "Generate Report"
- **Expected Result**: Error message "Failed to generate report ({status code})" is displayed. Button returns to "Generate Report" state.
- **Requires Human**: Yes

### TC-07-06: Report — no baby selected
- **Preconditions**: User has no active baby
- **Steps**:
  1. Navigate to /report
- **Expected Result**: The report page functionality depends on having an active baby (handled by the route page component)
- **Requires Human**: Yes

---

## 8. Navigation & Layout

### TC-08-01: NavHeader links — authenticated user with babies
- **Preconditions**: User is logged in with at least one baby
- **Steps**:
  1. Observe the navigation header
- **Expected Result**: Header contains: "LittleLiver" link (to /), BabySelector, "Trends" link (to /trends), "Report" link (to /report), "Medications" link (to /medications), "Settings" link (to /settings), "Logout" button
- **Requires Human**: Yes

### TC-08-02: NavHeader links — authenticated user without babies
- **Preconditions**: User is logged in, has no babies
- **Steps**:
  1. Observe the navigation header
- **Expected Result**: Header shows "LittleLiver" link, "Settings" link, and "Logout" button. BabySelector, Trends, Report, and Medications links are NOT shown (they are inside the `{#if $babies.length > 0}` block).
- **Requires Human**: Yes

### TC-08-03: NavHeader — LittleLiver link navigates home
- **Preconditions**: User is logged in, on any page
- **Steps**:
  1. Click "LittleLiver" in the header
- **Expected Result**: User is navigated to /
- **Requires Human**: Yes

### TC-08-04: NavHeader — baby selector in header
- **Preconditions**: User has multiple babies
- **Steps**:
  1. Change the baby selector dropdown in the header
- **Expected Result**: Active baby changes. Dashboard and all views update accordingly.
- **Requires Human**: Yes

### TC-08-05: Back link from metric forms
- **Preconditions**: On any /log/[metric] page
- **Steps**:
  1. Click the "Back" link
- **Expected Result**: Navigates to /
- **Requires Human**: Yes

### TC-08-06: Back link from medications page
- **Preconditions**: On /medications
- **Steps**:
  1. Click the "Back" link
- **Expected Result**: Navigates to /
- **Requires Human**: Yes

### TC-08-07: PWA — manifest and service worker registration
- **Preconditions**: App is loaded
- **Steps**:
  1. Open browser DevTools > Application tab
  2. Check for Service Worker registration
  3. Check for manifest.json link
- **Expected Result**: A service worker is registered. manifest.json is linked in the HTML head. Theme color meta tag is set to "#4a9c5e".
- **Requires Human**: Yes

### TC-08-08: PWA — install prompt
- **Preconditions**: App meets PWA installability criteria, viewed in a supporting browser
- **Steps**:
  1. Load the app
  2. Observe if browser shows install prompt (or check via setupInstallPrompt)
- **Expected Result**: The app captures the beforeinstallprompt event for potential install prompting
- **Requires Human**: Yes

### TC-08-09: Push notifications initialization
- **Preconditions**: User is authenticated
- **Steps**:
  1. Log in to the app
  2. Observe push notification initialization
- **Expected Result**: When the user is authenticated (currentUser is set), initPushNotifications() is called via the $effect in the layout
- **Requires Human**: Yes

---

## 9. Edge Cases & Error States

### TC-09-01: Empty dashboard — no data for today
- **Preconditions**: User has a baby but no entries logged today
- **Steps**:
  1. Navigate to /
- **Expected Result**: Summary cards show zeros and dashes. No stool trend section. No upcoming meds section (if no medications configured). Quick log buttons are still displayed.
- **Requires Human**: Yes

### TC-09-02: Empty medication list
- **Preconditions**: Baby has no medications
- **Steps**:
  1. Navigate to /medications
- **Expected Result**: "No medications found." message is displayed. "Add Medication" button is still available.
- **Requires Human**: Yes

### TC-09-03: Network error on dashboard load
- **Preconditions**: User is logged in, network is disconnected
- **Steps**:
  1. Disconnect network
  2. Navigate to /
- **Expected Result**: "Failed to load dashboard data" error message is displayed
- **Requires Human**: Yes

### TC-09-04: Network error on medication list load
- **Preconditions**: User is on /medications, network is disconnected
- **Steps**:
  1. Disconnect network, reload page
- **Expected Result**: "Failed to load medications" error is displayed
- **Requires Human**: Yes

### TC-09-05: Network error on medication toggle
- **Preconditions**: User is on /medications, then network drops
- **Steps**:
  1. Click "Deactivate" on a medication after network disconnect
- **Expected Result**: "Failed to update medication" error is displayed. Medication state is unchanged.
- **Requires Human**: Yes

### TC-09-06: Backdated entry
- **Preconditions**: On /log/feeding
- **Steps**:
  1. Change the timestamp to a date several days in the past
  2. Fill in required fields and submit
- **Expected Result**: Entry is saved with the backdated timestamp. Dashboard today totals are NOT affected (entry is in the past). Trends view shows the entry on the correct historical date.
- **Requires Human**: Yes

### TC-09-07: Rapid form submission (double-click prevention)
- **Preconditions**: On any metric entry form
- **Steps**:
  1. Fill in required fields
  2. Click submit rapidly twice
- **Expected Result**: Submit button becomes disabled (disabled={submitting}) after the first click, preventing duplicate submissions. Only one entry is created.
- **Requires Human**: Yes

### TC-09-08: QuickLogButtons — "More Entries" toggle
- **Preconditions**: On the dashboard (/)
- **Steps**:
  1. Observe the quick log section: 5 primary buttons visible (Feed, Wet Diaper, Stool, Temp, Medication Given)
  2. Click "More Entries"
  3. Observe additional buttons
  4. Click "Less Entries"
- **Expected Result**: "More Entries" reveals 7 additional buttons: Weight, Abdomen, Skin, Bruising, Lab, Notes, Manage Medications. Button text changes to "Less Entries". Clicking "Less Entries" hides the extra buttons and text reverts.
- **Requires Human**: Yes

### TC-09-09: QuickLogButtons — "Manage Medications" navigates to /medications
- **Preconditions**: On the dashboard, "More Entries" is expanded
- **Steps**:
  1. Click "Manage Medications"
- **Expected Result**: User is navigated to /medications
- **Requires Human**: Yes

### TC-09-10: Photo upload — file type restriction
- **Preconditions**: On a form with photo upload
- **Steps**:
  1. Attempt to select a non-image file (e.g., .txt, .pdf)
- **Expected Result**: Browser's file picker filters to only show accepted types (JPEG, PNG, HEIC). If somehow bypassed, the file input's accept attribute restricts selection.
- **Requires Human**: Yes (file selection)

### TC-09-11: Photo upload — disabled when max reached
- **Preconditions**: On a form with photo upload, 4 photos already uploaded
- **Steps**:
  1. Observe the file input
- **Expected Result**: File input is disabled. Photo count shows "4/4 photos".
- **Requires Human**: Yes

### TC-09-12: Edit medication — load failure
- **Preconditions**: Navigate to /log/medication?edit={invalid_id}
- **Steps**:
  1. Navigate to the URL
- **Expected Result**: Error message "Failed to load medication" is displayed after the load attempt fails
- **Requires Human**: Yes

---

## 10. Cross-Cutting Concerns

### TC-10-01: CSRF token fetched before state-changing requests
- **Preconditions**: User is logged in
- **Steps**:
  1. Open browser DevTools > Network tab
  2. Submit any form (e.g., log a feeding)
  3. Observe the requests
- **Expected Result**: Before the POST request, a GET to /api/csrf-token occurs (if not already cached). The POST request includes an X-CSRF-Token header.
- **Requires Human**: Yes

### TC-10-02: CSRF token cached after first fetch
- **Preconditions**: User is logged in
- **Steps**:
  1. Submit a form (CSRF token is fetched and cached)
  2. Submit another form
  3. Observe network requests
- **Expected Result**: Only one /api/csrf-token request is made. The second submission reuses the cached token.
- **Requires Human**: Yes

### TC-10-03: Timezone header sent with all requests
- **Preconditions**: User is logged in
- **Steps**:
  1. Open DevTools > Network tab
  2. Trigger any API request (e.g., load dashboard)
  3. Inspect the request headers
- **Expected Result**: Every API request includes an X-Timezone header with the user's timezone (e.g., "America/New_York")
- **Requires Human**: Yes

### TC-10-04: Session expiry redirects to login
- **Preconditions**: User is logged in, session has expired (or API returns 401)
- **Steps**:
  1. Wait for session to expire
  2. Trigger any API request
- **Expected Result**: The 401 response handler clears the CSRF token cache and redirects the browser to /login via window.location.href
- **Requires Human**: Yes

### TC-10-05: CSRF token cleared on 401
- **Preconditions**: A 401 response is received from the API
- **Steps**:
  1. Trigger a request that returns 401
  2. After redirect to /login, log back in
  3. Trigger a state-changing request
- **Expected Result**: A fresh CSRF token is fetched (the old cached token was cleared on the 401)
- **Requires Human**: Yes

### TC-10-06: Logout sends CSRF token
- **Preconditions**: User is logged in
- **Steps**:
  1. Click "Logout"
  2. Observe the network request
- **Expected Result**: POST to /auth/logout includes X-CSRF-Token and X-Timezone headers, and sends with credentials: 'include'
- **Requires Human**: Yes

### TC-10-07: Content-Type header set for JSON body
- **Preconditions**: User is logged in
- **Steps**:
  1. Submit a form with JSON body (e.g., log a feeding)
  2. Inspect the request headers
- **Expected Result**: The request includes Content-Type: application/json header (set automatically when body is a string)
- **Requires Human**: Yes

### TC-10-08: Content-Type NOT set for FormData body (photo upload)
- **Preconditions**: User is logged in
- **Steps**:
  1. Upload a photo on any form that supports it
  2. Inspect the request headers
- **Expected Result**: The request does NOT have Content-Type: application/json. The browser sets the appropriate multipart/form-data Content-Type automatically for FormData.
- **Requires Human**: Yes

### TC-10-09: 204 No Content response handled
- **Preconditions**: User is logged in, API returns 204 for a request
- **Steps**:
  1. Perform an action where the API returns 204 (e.g., delete operation)
- **Expected Result**: No JSON parse error occurs. The response is returned as undefined.
- **Requires Human**: Yes

### TC-10-10: Google OAuth link uses data-sveltekit-reload
- **Preconditions**: On /login page
- **Steps**:
  1. Inspect the "Sign in with Google" link element
- **Expected Result**: The link has href="/auth/google/login" and the data-sveltekit-reload attribute, ensuring a full page navigation (not a SvelteKit client-side navigation) to the OAuth endpoint
- **Requires Human**: No
