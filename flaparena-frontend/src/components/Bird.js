import birdImageSrc from '../assets/images/bird.png';

class Bird {
  static image = new Image();

  static loadImage() {
    Bird.image.src = birdImageSrc;
    return new Promise((resolve) => {
      Bird.image.onload = () => resolve();
    });
  }

  static draw(ctx, position) {
    ctx.drawImage(Bird.image, position.x, position.y);
  }
}

export default Bird;
