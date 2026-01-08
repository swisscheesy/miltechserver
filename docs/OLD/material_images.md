You will be creating a task list to implement a new feature.  This feature is to allow logged-in users to upload images to the azure container 'material-images', that will be assigned to specific item NIINS.  Once the images are uploaded, they can be queried from the server by a specific NIIN and be shown to the user.  This will require implementing the new feature in the server code, as well as creating the postgres schema for the database.  Here is a basic overview of what the users will be able to do once the feature is fully implemented.

1.  Logged-in users will be allowed to upload images for a specific NIIN.
    1. The user must be logged in to upload an image.
    2. Only the user that uploaded the image is allowed to delete it from the container/database.
    3. Users are only able to upload 1 image in a 1 hour period for a single niin, but they can upload images to multiple niins in a 1 hour period.

2. All users are allowed to view the item images.

3. There needs to be a downvote/flag system in place so users can downvote or flag a specific image for correctness and removal.
    1. Only logged in users are allowed to downvote/flag images.

4. The database needs to track the downvote/flag system, and the saved location in the container for each image.

5. A niin can have multiple images.

The complete routes need to be created and implemented, this includes the routes, repository, service, and controller implementations.

Analyze how the current azure container for saving item images is implemented if you need to understand an implementation.  
The feature files should be created with the 'material_images' prefix.  Take your time to fully flesh out all tasks that will be required to implement this feature, as well as schema changes.  Be as detailed as possible and store the results in a .md file for future review and usage.  Include all required schema changes.  Make no project changes at this time, only analyze, plan and create results in a .md