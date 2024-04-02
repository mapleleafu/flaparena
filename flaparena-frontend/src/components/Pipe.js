class PipePair {
  constructor(x, gapTop, gapSize, pipeImage) {
    this.x = x;
    this.gapTop = gapTop;
    this.gapSize = gapSize;
    this.pipeImage = pipeImage;
  }

  draw(ctx) {
    const pipeWidth = this.pipeImage.width; // Use image width
    const canvasHeight = ctx.canvas.height;

    // Draw top pipe (flipped vertically)
    ctx.save();
    ctx.translate(this.x + pipeWidth / 2, 0); // Translate to the center of the top pipe
    ctx.scale(1, -1); // Flip the context vertically
    ctx.drawImage(this.pipeImage, -pipeWidth / 2, -this.gapTop, pipeWidth, this.gapTop);
    ctx.restore();

    // Draw bottom pipe
    const bottomPipeY = this.gapTop + this.gapSize;
    ctx.drawImage(this.pipeImage, this.x, bottomPipeY, pipeWidth, canvasHeight - bottomPipeY);
  }

  update() {
    this.x -= 2; // Move pipes to the left
  }
}

export default PipePair;
